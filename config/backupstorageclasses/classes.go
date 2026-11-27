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

// Package backupstorageclasses embeds the built-in BackupStorageClasses as
// make gen writes them, contract version stamped, so the controller can
// apply them where the installer cannot ship them, such as an OLM bundle.
package backupstorageclasses

import "embed"

// Manifests holds the classes and kustomization.yaml, whose resources list
// the files that hold them.
//
//go:embed *.yaml
var Manifests embed.FS
