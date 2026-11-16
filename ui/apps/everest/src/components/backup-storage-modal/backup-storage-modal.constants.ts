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

import { FieldType, Section } from 'components/ui-generator/ui-generator.types';
import { SectionFormContract } from 'components/ui-generator/utils/section-form';
import { Messages } from './backup-storage-modal.messages';

// Keys the class's location and storage forms are prepared under.
export const LOCATION_SECTION_KEY = 'location';
export const STORAGE_SECTION_KEY = 'storage';

// The fields of the class's authentication and of TLS are those of
// AuthenticationFormFields and TlsConfigFormFields.
export enum StorageFormFields {
  name = 'name',
  namespace = 'namespace',
  className = 'className',
  // Root of the class's location and storage forms, whose paths are relative
  // to the storage spec (e.g. location.objectStore.bucket).
  spec = 'spec',
  // Root of the form generated from parametersSchema when the class ships no
  // storage form.
  fallbackParameters = 'fallbackParameters',
}

const BUCKET_PATH = 'location.objectStore.bucket';
const PREFIX_PATH = 'location.objectStore.prefix';

// Every storage's location is a bucket and a prefix in an object store; this
// form asks for them when the class ships no location form.
export const DEFAULT_LOCATION_SECTION: Section = {
  components: {
    bucket: {
      uiType: FieldType.Text,
      path: BUCKET_PATH,
      fieldParams: { label: Messages.bucket },
      validation: { required: true },
    },
    prefix: {
      uiType: FieldType.Text,
      path: PREFIX_PATH,
      fieldParams: {
        label: Messages.prefix,
        helperText: Messages.prefixHelper,
      },
    },
  },
  componentsOrder: ['bucket', 'prefix'],
};

// What the BackupStorage CRD asks of spec.location, whichever form fills it.
export const LOCATION_CONTRACT: SectionFormContract = {
  path: 'location',
  schema: {
    type: 'object',
    required: ['objectStore'],
    properties: {
      objectStore: {
        type: 'object',
        required: ['bucket'],
        properties: {
          bucket: {
            type: 'string',
            minLength: 1,
            maxLength: 255,
            pattern: '^[^/]+$',
          },
          prefix: {
            type: 'string',
            maxLength: 512,
            pattern: '^([^/]+(/[^/]+)*)?$',
          },
        },
      },
    },
  },
};
