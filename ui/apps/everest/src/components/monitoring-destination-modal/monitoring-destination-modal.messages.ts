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
  createTitle: 'Add monitoring destination',
  editTitle: 'Edit monitoring destination',
  create: 'Add',
  save: 'Save',
  name: 'Name',
  nameFieldName: 'name',
  namespace: 'Namespace',
  monitoringClass: 'Type',
  required: 'Required',
  authenticationNouns: {
    object: 'destination',
    class: 'monitoring class',
  } satisfies AuthenticationNouns,
  changedMeanwhile:
    'This destination was changed meanwhile. The form now shows the latest version with your changes kept; review it and save again.',
  restartsInstances: (count: number) => {
    const instances =
      count === 1
        ? 'the instance that sends to this destination'
        : `the ${count} instances that send to this destination`;
    return `Saving these changes may restart the pods of ${instances}, one at a time; an instance with a single replica may be briefly unavailable.`;
  },
};
