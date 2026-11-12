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

package main

import (
	"context"
	"fmt"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/openeverest/openeverest/v2/internal/controller/secretusage"
)

// secretProtectionFlag turns the in-use protection of Secrets on or off; the
// chart sets it from secretProtection.enabled.
const secretProtectionFlag = "secret-protection"

// releaseRetry paces the retries of releaseSecrets: the delay doubles from
// first up to most.
type releaseRetry struct {
	first, most time.Duration
}

const (
	releaseFirstRetry = time.Second
	releaseMostRetry  = 5 * time.Minute
)

// setupSecretProtection registers the Secret usage reconciler for
// referrers when protection is enabled. When it is not, it registers
// instead a one-off release of what an earlier run held on Secrets, their
// finalizer and usage counts, and no reconciler puts them back.
func setupSecretProtection(mgr ctrl.Manager, enabled bool, referrers []secretusage.Referrer) error {
	if enabled {
		return (&secretusage.Reconciler{
			Client:    mgr.GetClient(),
			APIReader: mgr.GetAPIReader(),
			Recorder:  mgr.GetEventRecorder("openeverest-secretusage"),
			Referrers: referrers,
		}).SetupWithManager(mgr)
	}
	// Uncached: a metadata-only list through the manager's client would
	// start an informer on every Secret in the cluster.
	c, err := client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme(), Mapper: mgr.GetRESTMapper()})
	if err != nil {
		return fmt.Errorf("create the release client: %w", err)
	}
	return mgr.Add(releaseSecrets(c, referrers, releaseRetry{first: releaseFirstRetry, most: releaseMostRetry}))
}

// releaseSecrets removes core's finalizer and the usage counts of
// referrers from every Secret, retrying until it succeeds or the manager
// stops; a failure does not stop the manager. As a runnable that needs
// leader election, it runs only on the leader, so a previous leader with
// protection enabled has stopped before it starts, and nothing puts the
// marks back.
func releaseSecrets(c client.Client, referrers []secretusage.Referrer, retry releaseRetry) manager.RunnableFunc {
	return func(ctx context.Context) error {
		log := logf.FromContext(ctx).WithName("release-secret-protection")
		delay := retry.first
		for {
			released, err := secretusage.Release(ctx, c, referrers)
			if err == nil {
				log.Info("Released the in-use protection of Secrets, which is disabled", "secrets", released)
				return nil
			}
			log.Error(err, "Failed to release the in-use protection of Secrets; retrying", "after", delay)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(delay):
			}
			delay = min(2*delay, retry.most)
		}
	}
}
