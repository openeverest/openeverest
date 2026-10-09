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

import { useEffect } from 'react';
import { useFormContext } from 'react-hook-form';
import { TextInput } from '@percona/ui-lib';
import { SECRET_NAME_FIELD } from '../secret-create-modal.constants';
import { Messages } from '../secret-create-modal.messages';
import { SecretNameInputProps } from './secret-name-input.types';

export const SecretNameInput = ({ takenNames }: SecretNameInputProps) => {
  const { trigger } = useFormContext();

  // A name the server just rejected is flagged without waiting for a keystroke.
  useEffect(() => {
    if (takenNames.length > 0) trigger(SECRET_NAME_FIELD);
  }, [takenNames, trigger]);

  return <TextInput name={SECRET_NAME_FIELD} label={Messages.nameLabel} />;
};
