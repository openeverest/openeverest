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

import { AuthenticationNouns } from 'components/class-authentication';

export const Messages = {
  createTitle: 'Add backup storage',
  editTitle: 'Edit backup storage',
  create: 'Add',
  save: 'Save',
  name: 'Name',
  nameFieldName: 'name',
  namespace: 'Namespace',
  storageClass: 'Type',
  required: 'Required',
  bucket: 'Bucket',
  prefix: 'Prefix',
  prefixHelper: 'Optional path within the bucket',
  fixedAfterCreation:
    "The location can't change once the storage is added. Add another storage instead.",
  authenticationNouns: {
    object: 'backup storage',
    class: 'backup storage type',
  } satisfies AuthenticationNouns,
  classUnavailable: (className: string) =>
    `This storage's type, ${className}, is no longer available, so the storage can't be edited.`,
  changedMeanwhile:
    'This storage was changed meanwhile. The form now shows the latest version with your changes kept; review it and save again.',
};
