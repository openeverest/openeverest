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

import { MenuItem } from '@mui/material';
import { SelectInput } from '@percona/ui-lib';
import { ClassContract } from 'utils/class-contract';
import { AuthenticationFormFields } from '../class-authentication.constants';
import { Messages } from '../class-authentication.messages';

interface AuthModeSelectProps {
  contract: ClassContract;
}

export const AuthModeSelect = ({ contract }: AuthModeSelectProps) => (
  <SelectInput
    name={AuthenticationFormFields.authMode}
    label={Messages.authMode}
    isRequired
  >
    {contract.authModes.map(({ name }) => (
      <MenuItem key={name} value={name}>
        {name}
      </MenuItem>
    ))}
  </SelectInput>
);
