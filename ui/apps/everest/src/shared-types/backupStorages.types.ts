// Copyright (C) 2023 Percona LLC
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

import { CrdsGen, HttpApi } from '@generated/api-types';
import { Section } from 'components/ui-generator/ui-generator.types';

/** The full CRD type returned by the v2 cluster-scoped API. */
export type BackupStorageCRD = CrdsGen.components['schemas']['BackupStorage'];
export type BackupStorageListCRD =
  CrdsGen.components['schemas']['BackupStorageList'];
export type BackupStorageListFilter = NonNullable<
  HttpApi.operations['listBackupStorages']['parameters']['query']
>;
export type BackupStorageCondition = NonNullable<
  NonNullable<BackupStorageCRD['status']>['conditions']
>[number];

export type BackupStorageClass =
  CrdsGen.components['schemas']['BackupStorageClass'];
export type BackupStorageClassList =
  CrdsGen.components['schemas']['BackupStorageClassList'];
export type BackupStorageClassAuthMode =
  BackupStorageClass['spec']['authModes'][number];

// The generated API types define uiSchema as `Record<string, never>` (opaque).
export type BackupStorageClassUiSchema = {
  location?: Section;
  storage?: Section;
  credentials?: Record<string, Section>;
};

export enum CredentialsSource {
  Existing = 'existing',
  New = 'new',
}

export enum TlsMode {
  System = 'system',
  CustomCA = 'customCA',
  SkipVerify = 'skipVerify',
}

export enum CaSource {
  New = 'new',
  Existing = 'existing',
}

// Also the suffix of the Secret the form creates for a storage.
export enum BackupStorageSecretKind {
  Credentials = 'credentials',
  CA = 'ca',
}

/**
 * Form-values representation of a backup storage.
 * Used only in create/edit forms and as the input to mutation hooks.
 */
export interface BackupStorageFormValues {
  name: string;
  namespace: string;
  backupStorageClass: string;
  // Fields rendered from the class's uiSchema, at paths relative to the spec.
  spec: {
    location?: Partial<BackupStorageCRD['spec']['location']>;
    parameters?: Record<string, unknown>;
  };
  authMode: string;
  credentialsSource: CredentialsSource;
  credentialsSecretName: string;
  // Keys of the auth mode's credentials, rendered from the class's uiSchema.
  credentials: Record<string, unknown>;
  tlsMode: TlsMode;
  caSource: CaSource;
  caBundle: string;
  caSecretName: string;
  caSecretKey: string;
}
