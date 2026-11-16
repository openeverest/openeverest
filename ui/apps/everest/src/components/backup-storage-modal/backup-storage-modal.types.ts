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

import { AuthenticationFormValues } from 'components/class-authentication';
import { OpenApiParameterFields } from 'components/openapi-parameters-fields';
import { TlsConfigFormValues } from 'components/tls-config-fields';
import { SectionForm } from 'components/ui-generator/utils/section-form';
import {
  BackupStorageClass,
  BackupStorageCRD,
} from 'shared-types/backupStorages.types';

export interface BackupStorageFormValues
  extends AuthenticationFormValues,
    TlsConfigFormValues {
  name: string;
  namespace: string;
  className: string;
  spec: Record<string, unknown>;
  fallbackParameters: Record<string, unknown>;
}

// The forms a class renders for a storage, for the chosen auth mode.
export interface StorageClassForms {
  location: SectionForm;
  storage?: SectionForm;
  // Used when the class ships no storage form.
  parameterFields: OpenApiParameterFields;
  credentials?: SectionForm;
}

export interface BackupStorageTarget {
  namespace: string;
  name: string;
}

export interface BackupStorageModalProps {
  // The storage to edit; a new one is added without it.
  target?: BackupStorageTarget;
  // The namespace a new storage starts in.
  namespace?: string;
  onClose: () => void;
}

export interface BackupStorageFormProps
  extends Omit<BackupStorageModalProps, 'target'> {
  original?: BackupStorageCRD;
  classes: BackupStorageClass[];
  // Namespaces the user may add storages in.
  namespaces: string[];
  // Set after a save was refused because the storage changed meanwhile.
  changedMeanwhile: boolean;
  onChangedMeanwhile: () => void;
}

// The namespaces the user may add storages in.
export interface BackupStoragePermissions {
  createNamespaces: string[];
}
