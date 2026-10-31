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

// Package storageclasses holds the Go types of the built-in
// BackupStorageClasses, the decoders that turn a BackupStorage into them, and
// Accept, the verdict core publishes as the storage's Accepted condition.
//
// The parametersSchema of every built-in class is generated from these types
// (make gen writes config/backupstorageclasses), so the schema a storage is
// validated against and the decoder executors read it with cannot drift.
// Every field of a class parameter type is a connection parameter and carries
// the +openeverest:param=connection marker; the location (bucket, prefix) and
// transport trust (tls) are typed on BackupStorage instead.
package storageclasses
