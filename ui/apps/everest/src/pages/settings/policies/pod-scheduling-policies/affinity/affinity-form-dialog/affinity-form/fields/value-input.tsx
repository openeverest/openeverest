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

import { TextInput } from '@percona/ui-lib';
import { SxProps, Theme } from '@mui/material';
import { AffinityFormFields } from '../affinity-form.types';

type Props = {
  disabled: boolean;
  namePrefix?: string;
  sx?: SxProps<Theme>;
};

const ValueInput = ({ disabled, namePrefix = '', sx }: Props) => (
  <TextInput
    name={`${namePrefix}${AffinityFormFields.values}`}
    label={'Values'}
    textFieldProps={{
      sx: sx ?? {
        marginTop: '25px',
        width: '645px',
      },
      inputProps: {
        disabled,
      },
      helperText: 'Insert comma seperated values',
    }}
  />
);

export default ValueInput;
