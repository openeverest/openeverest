// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package minio implements a provider-runtime ProviderInterface for MinIO
// object storage. This is a Phase 0-3 PoC (LFX Term 3, upstream issue
// openeverest#2255): Sync renders and applies a real MinIO Operator Tenant
// CR (minio.min.io/v1) from the Instance spec, and Status reads it back and,
// once Ready, registers the Tenant as a BackupStorage other
// OpenEverest-managed databases can use as a backup target. Dedicated
// bucket/user/policy management (the open design question in
// design-notes.md) is deliberately out of scope here.
package minio

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	commonv1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// serverComponentType is the only componentType this Phase 2 PoC knows
// about, matching manifest/provider.yaml.
const serverComponentType = "server"

// tenantStatusInitialized is the MinIO Operator's Tenant.status.currentState
// value once the Tenant is fully up. Verified directly against a running
// operator during Phase 0/1 (see design-notes.md).
const tenantStatusInitialized = "Initialized"

// defaultMinIOImage is the operator's own fallback image, used only if the
// Provider CR somehow declares no default version for the "server"
// componentType. Matches github.com/minio/operator/pkg/apis/minio.min.io/v1.DefaultMinIOImage.
const defaultMinIOImage = "minio/minio:RELEASE.2020-12-23T02-24-12Z"

// defaultVolumeSize is used when the Instance's "server" component doesn't
// specify storage. Matches the size hand-verified against a real kind
// cluster in Phase 0.
var defaultVolumeSize = resource.MustParse("1Gi")

// rootUser is the MinIO root user name written into the generated
// credentials Secret. MinIO accepts any value here; the operator itself has
// no default, so a fixed name is used for this PoC (bucket/user management
// is a deferred design question, see design-notes.md).
const rootUser = "minio"

// configSecretSuffix names the Secret holding the Tenant's root credentials,
// referenced via Tenant.Spec.Configuration.Name. Discovered the hard way in
// Phase 2: the MinIO Operator leaves a Tenant stuck at
// status.currentState == "empty tenant credentials" indefinitely if this
// Secret (and the spec.configuration.name reference to it) is missing —
// confirmed against a real Tenant applied without it, see design-notes.md.
const configSecretSuffix = "-env-configuration"

// backupBucketSuffix names the bucket auto-provisioned on the Tenant for the
// Phase 3 backup bridge: on Ready, this Instance's own Tenant is registered
// as a BackupStorage other OpenEverest-managed databases can back up into.
const backupBucketSuffix = "-backups"

// backupCredsSecretSuffix names the Secret backing the BackupStorage's
// CredentialsSecretRef. Kept separate from the Tenant's own
// configSecretSuffix Secret because the two use different key conventions
// (config.env shell-export lines vs. AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY,
// the keys BackupStorageS3Spec.CredentialsSecretRef documents and
// controller.Context.BackupStorageCredentials reads).
const backupCredsSecretSuffix = "-backup-credentials"

// backupStorageRegion is a fixed placeholder: MinIO doesn't have regions,
// but BackupStorageS3Spec.Region is required (S3-compatible SDKs need
// something in the field). Matches the convention already used for MinIO in
// this repo's own dev/resources/backupstorage.yaml fixture.
const backupStorageRegion = "us-east-1"

// Provider implements controller.ProviderInterface for MinIO.
type Provider struct{}

func (p *Provider) Name() string { return "minio" }

// Types registers this package's Tenant/TenantList scheme additions, needed
// so the runtime's client can Get/Apply/own Tenant objects.
func (p *Provider) Types() func(*runtime.Scheme) error { return AddToScheme }

func (p *Provider) Validate(c *controller.Context) error {
	log.FromContext(c.Context()).Info("minio provider: Validate called",
		"instance", c.Name(), "namespace", c.Namespace())
	return nil
}

// Sync renders a MinIO Operator Tenant CR from the Instance spec and applies
// it. The "server" component (if present) maps onto the Tenant's single
// Pool: Replicas -> Pool.Servers, Storage -> Pool.VolumeClaimTemplate.
// VolumesPerServer is hardcoded to 1 for this PoC (see design-notes.md's
// open question on modeling it distinctly from replica count).
func (p *Provider) Sync(c *controller.Context) error {
	logger := log.FromContext(c.Context())
	logger.Info("minio provider: Sync called", "instance", c.Name(), "namespace", c.Namespace())

	providerSpec, err := c.ProviderSpec()
	if err != nil {
		return fmt.Errorf("minio provider: fetching provider spec: %w", err)
	}

	replicas := int32(1)
	size := defaultVolumeSize
	var storageClass *string
	version := ""

	if servers := c.ComponentsOfType(serverComponentType); len(servers) > 0 {
		server := servers[0]
		version = server.Version
		if server.Replicas != nil {
			replicas = *server.Replicas
		}
		if server.Storage != nil {
			if !server.Storage.Size.IsZero() {
				size = server.Storage.Size
			}
			storageClass = server.Storage.StorageClass
		}
	}

	image := resolveServerImage(providerSpec, version)

	creds, err := ensureCredentialsSecret(c)
	if err != nil {
		return fmt.Errorf("minio provider: ensuring credentials secret: %w", err)
	}

	tenant := &Tenant{
		ObjectMeta: c.ObjectMeta(c.Name()),
		Spec: TenantSpec{
			Image: image,
			// false, not nil/true: the CRD declares no default for this
			// field, and Phase 0's hand-verified working Tenant set it to
			// false explicitly (avoids depending on cert-manager or the
			// operator's own CSR-based autocert flow for this PoC).
			RequestAutoCert: new(bool),
			Configuration:   &corev1.LocalObjectReference{Name: creds.secretName},
			Buckets:         []Bucket{{Name: c.Name() + backupBucketSuffix}},
			Pools: []Pool{
				{
					Name:             "pool-0",
					Servers:          replicas,
					VolumesPerServer: 1,
					VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{Name: "data"},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{corev1.ResourceStorage: size},
							},
							StorageClassName: storageClass,
						},
					},
				},
			},
		},
	}

	if err := c.Apply(tenant); err != nil {
		return fmt.Errorf("minio provider: applying Tenant %q: %w", tenant.Name, err)
	}
	return nil
}

// credentials is the resolved MinIO root user/password backing a Tenant's
// credentials Secret, along with that Secret's own name.
type credentials struct {
	secretName   string
	rootUser     string
	rootPassword string
}

// ensureCredentialsSecret returns the credentials backing the Tenant's
// spec.configuration.name, creating them with a freshly generated root
// password if the Secret doesn't already exist. It deliberately does not
// use c.Apply for the already-exists case: c.Apply always issues an Update,
// which would overwrite the password on every reconcile and desync it from
// whatever the MinIO server actually booted with (the operator only reads
// this Secret once, at Tenant bootstrap). The plain "rootUser"/"rootPassword"
// keys (alongside the config.env the Tenant itself needs) exist so callers
// don't have to re-parse the shell-export format to recover the password —
// used by ensureBackupStorage to mint S3-style credentials for the same
// account.
func ensureCredentialsSecret(c *controller.Context) (credentials, error) {
	name := c.Name() + configSecretSuffix

	existing := &corev1.Secret{}
	exists, err := c.Exists(existing, name)
	if err != nil {
		return credentials{}, fmt.Errorf("checking for existing credentials secret %q: %w", name, err)
	}
	if exists {
		return credentials{
			secretName:   name,
			rootUser:     string(existing.Data["rootUser"]),
			rootPassword: string(existing.Data["rootPassword"]),
		}, nil
	}

	password, err := randomHex(20)
	if err != nil {
		return credentials{}, fmt.Errorf("generating root password: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: c.ObjectMeta(name),
		Data: map[string][]byte{
			"config.env":   fmt.Appendf(nil, "export MINIO_ROOT_USER=%q\nexport MINIO_ROOT_PASSWORD=%q\n", rootUser, password),
			"rootUser":     []byte(rootUser),
			"rootPassword": []byte(password),
		},
	}
	if err := c.Apply(secret); err != nil {
		return credentials{}, fmt.Errorf("creating credentials secret %q: %w", name, err)
	}
	return credentials{secretName: name, rootUser: rootUser, rootPassword: password}, nil
}

// ensureBackupStorage registers this Instance's own Tenant as a
// BackupStorage other OpenEverest-managed databases can back up into — the
// Phase 3 "backup bridge" (see ROADMAP.md §4 Phase 3, design-notes.md). It
// reuses the Tenant's root credentials rather than minting a dedicated MinIO
// user/policy: bucket/user/policy management is the open design question in
// design-notes.md, deliberately deferred past this PoC.
func ensureBackupStorage(c *controller.Context) error {
	creds, err := ensureCredentialsSecret(c)
	if err != nil {
		return fmt.Errorf("resolving root credentials: %w", err)
	}

	backupSecretName := c.Name() + backupCredsSecretSuffix
	backupSecret := &corev1.Secret{
		ObjectMeta: c.ObjectMeta(backupSecretName),
		Data: map[string][]byte{
			"AWS_ACCESS_KEY_ID":     []byte(creds.rootUser),
			"AWS_SECRET_ACCESS_KEY": []byte(creds.rootPassword),
		},
	}
	if err := c.Apply(backupSecret); err != nil {
		return fmt.Errorf("applying backup credentials secret %q: %w", backupSecretName, err)
	}

	verifyTLS := false
	forcePathStyle := true
	backupStorage := &backupv1alpha1.BackupStorage{
		ObjectMeta: c.ObjectMeta(c.Name()),
		Spec: backupv1alpha1.BackupStorageSpec{
			Type: backupv1alpha1.BackupStorageTypeS3,
			S3: &backupv1alpha1.BackupStorageS3Spec{
				Bucket: c.Name() + backupBucketSuffix,
				Region: backupStorageRegion,
				// In-cluster headless Service the MinIO Operator creates
				// for every Tenant, named "<tenant>-hl", port 9000 — the
				// same endpoint hand-verified with a real mc round-trip in
				// Phase 2 (see design-notes.md).
				EndpointURL:          fmt.Sprintf("http://%s-hl.%s.svc.cluster.local:9000", c.Name(), c.Namespace()),
				VerifyTLS:            &verifyTLS,
				ForcePathStyle:       &forcePathStyle,
				CredentialsSecretRef: commonv1alpha1.SecretRef{Name: backupSecretName},
			},
		},
	}
	if err := c.Apply(backupStorage); err != nil {
		return fmt.Errorf("applying BackupStorage %q: %w", backupStorage.Name, err)
	}
	return nil
}

// randomHex returns a cryptographically random hex string of n bytes (2n
// hex characters). Hex-only output is safe to embed unescaped inside the
// double-quoted shell-export lines config.env requires.
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// resolveServerImage looks up the image for the given resolved component
// version in the Provider's "server" componentType. Falls back to the
// componentType's default version's image, then to the operator's own
// built-in default image, if version is empty or unresolvable.
func resolveServerImage(spec *corev1alpha1.ProviderSpec, version string) string {
	ct, ok := spec.ComponentTypes[serverComponentType]
	if !ok {
		return defaultMinIOImage
	}

	var defaultImage string
	for _, v := range ct.Versions {
		if v.Version == version && version != "" {
			return v.Image
		}
		if v.Default {
			defaultImage = v.Image
		}
	}
	if defaultImage != "" {
		return defaultImage
	}
	return defaultMinIOImage
}

// Status reads the MinIO Operator Tenant's observed state back onto the
// Instance. tenantStatusInitialized is treated as Ready; anything else
// (including "not created yet") is Provisioning.
func (p *Provider) Status(c *controller.Context) (controller.Status, error) {
	logger := log.FromContext(c.Context())
	logger.Info("minio provider: Status called", "instance", c.Name(), "namespace", c.Namespace())

	tenant := &Tenant{}
	if err := c.Get(tenant, c.Name()); err != nil {
		if apierrors.IsNotFound(err) {
			return controller.Provisioning("waiting for MinIO Tenant to be created"), nil
		}
		return controller.Status{}, fmt.Errorf("minio provider: getting Tenant %q: %w", c.Name(), err)
	}

	desired := int32(1)
	if servers := c.ComponentsOfType(serverComponentType); len(servers) > 0 && servers[0].Replicas != nil {
		desired = *servers[0].Replicas
	}

	componentStatus := controller.ComponentStatus{
		Name:  serverComponentType,
		Ready: tenant.Status.AvailableReplicas,
		Total: desired,
		State: tenant.Status.CurrentState,
	}

	if tenant.Status.CurrentState != tenantStatusInitialized {
		status := controller.Provisioning(tenant.Status.CurrentState)
		status.Components = []controller.ComponentStatus{componentStatus}
		return status, nil
	}

	if err := ensureBackupStorage(c); err != nil {
		return controller.Status{}, fmt.Errorf("minio provider: registering BackupStorage: %w", err)
	}

	status := controller.Ready()
	status.Components = []controller.ComponentStatus{componentStatus}
	return status, nil
}

func (p *Provider) Cleanup(c *controller.Context) error {
	log.FromContext(c.Context()).Info("minio provider: Cleanup called",
		"instance", c.Name(), "namespace", c.Namespace())
	return nil
}
