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

// Command main is the entrypoint for the minio provider (Phase 1 skeleton).
// It follows spec 001's documented Provider Go SDK entry-point pattern.
package main

import (
	"os"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openeverest/openeverest/v2/provider-runtime/reconciler"
	minio "github.com/openeverest/openeverest/v2/providers/minio"
)

func main() {
	ctx := ctrl.SetupSignalHandler()

	p := &minio.Provider{}

	r, err := reconciler.New(ctx, p)
	if err != nil {
		log.Log.Error(err, "failed to create minio provider reconciler")
		os.Exit(1)
	}

	if err := r.Start(ctx); err != nil {
		log.Log.Error(err, "minio provider reconciler exited with error")
		os.Exit(1)
	}
}
