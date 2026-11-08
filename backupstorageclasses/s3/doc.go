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

// Package s3 is the contract of the s3 BackupStorageClass, for Amazon S3 and
// S3-compatible object stores: the Go types of its connection parameters, its
// authentication modes and their credentials keys, strict decoders and the
// generated class manifest. Core ships the class; every executor that writes
// to an s3 storage decodes it with this package.
//
// The class's parametersSchema is generated from these types (make gen writes
// class.yaml here and config/backupstorageclasses/s3.yaml), so the schema a
// storage is validated against and the decoder executors read it with cannot
// drift. Every field of Parameters is a connection parameter and carries the
// +openeverest:param=connection marker; the location (bucket, prefix) and
// transport trust (tls) are typed on BackupStorage instead.
package s3
