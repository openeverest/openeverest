package kubernetes

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openeverest/openeverest/v2/pkg/common"
)

const (
	privateKeyFile = "id_rsa"
	publicKeyFile  = "id_rsa.pub"

	keySize = 1024
)

func encodeRSAPrivateKey(key *rsa.PrivateKey) []byte {
	der := x509.MarshalPKCS1PrivateKey(key)
	block := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   der,
	}
	return pem.EncodeToMemory(&block)
}

func encodeRSAPublicKey(key *rsa.PublicKey) []byte {
	der := x509.MarshalPKCS1PublicKey(key)
	block := pem.Block{
		Type:    "RSA PUBLIC KEY",
		Headers: nil,
		Bytes:   der,
	}
	return pem.EncodeToMemory(&block)
}

func generatePrivateKey() (*rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to generate RSA key"))
	}
	if err := key.Validate(); err != nil {
		return nil, errors.Join(err, errors.New("failed to validate RSA key"))
	}
	return key, nil
}

// newRSAKeyPair generates a new RSA key pair.
// Returns: [publicKey, privateKey, error].
func newRSAKeyPair() ([]byte, []byte, error) {
	privateKey, err := generatePrivateKey()
	if err != nil {
		return nil, nil, err
	}
	publicKey := privateKey.Public().(*rsa.PublicKey) //nolint:forcetypeassert
	return encodeRSAPublicKey(publicKey), encodeRSAPrivateKey(privateKey), nil
}

// CreateRSAKeyPair creates a new RSA key pair and stores it in a secret.
func (k *Kubernetes) CreateRSAKeyPair(ctx context.Context) error {
	// Create a new key pair.
	publicKey, privateKey, err := newRSAKeyPair()
	if err != nil {
		return err
	}

	// Check if the secret exists?
	exists := false
	secret, err := k.GetSecret(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.EverestJWTSecretName})
	if err == nil {
		exists = true
	} else if !k8serrors.IsNotFound(err) {
		return err
	}

	// Create secret if it doesn't exists.
	if !exists {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.EverestJWTSecretName,
				Namespace: common.SystemNamespace,
			},
			Data: map[string][]byte{
				publicKeyFile:  publicKey,
				privateKeyFile: privateKey,
			},
		}
		if _, err := k.CreateSecret(ctx, secret); err != nil {
			return err
		}
		// Restart the deployment to pick up the new secret.
		return k.RestartDeployment(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.PerconaEverestDeploymentName})
	}

	// Otherwise, update the secret.
	secret.Data[publicKeyFile] = publicKey
	secret.Data[privateKeyFile] = privateKey
	if _, err := k.UpdateSecret(ctx, secret); err != nil {
		return err
	}
	// Restart the deployment to pick up the new secret.
	return k.RestartDeployment(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.PerconaEverestDeploymentName})
}
