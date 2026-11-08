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

// Package backupstorageclasses holds what the contracts of BackupStorageClasses
// share: the errors their decoders return, strict decoding of a
// BackupStorage's parameters and endpoint URL parsing. Each contract is a
// subpackage, such as s3 for the s3 class.
//
// A contract is used by every executor of its class, inside OpenEverest or
// not, so these packages import nothing from the rest of OpenEverest: they
// take a BackupStorage's location and parameters as plain values. A test
// enforces it, so a contract can move to a repository of its own unchanged.
package backupstorageclasses
