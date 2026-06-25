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

import { CreateEditStorageForm } from './create-edit-form';
import { validateInputWithRFC1035 } from 'utils/tests/validate-rfc1035';
import { storageLocationsSchema } from '../storage-locations.types';

vi.mock('hooks/api/namespaces/useNamespaces', () => ({
  useNamespaces: () => ({
    data: [],
    isFetching: false,
  }),
}));

const errors = {
  MIN1_ERROR: 'String must contain at least 1 character(s)',
  MAX22_ERROR: 'String must contain at most 22 character(s)',
  SPECIAL_CHAR_ERROR:
    'The storage name should only contain lowercase letters, numbers and hyphens.',
  START_CHAR_ERROR: "The name shouldn't start with a hyphen or a number.",
  END_CHAR_ERROR: "The name shouldn't end with a hyphen.",
};

validateInputWithRFC1035({
  renderComponent: () => <CreateEditStorageForm isEditMode={false} />,
  suiteName: 'Backup storage modal',
  errors: errors,
  schema: storageLocationsSchema,
});
