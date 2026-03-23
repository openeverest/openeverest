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

import { useMutation, UseMutationOptions } from '@tanstack/react-query';
import { createInstanceFn } from 'api/instanceApi';
import { DbWizardType } from 'pages/database-form/database-form-schema';

type CreateInstanceHookArgType = {
  formValue: DbWizardType;
};

export const useCreateInstance = (
  options?: UseMutationOptions<
    DbWizardType,
    unknown,
    CreateInstanceHookArgType,
    unknown
  >
) =>
  useMutation({
    mutationFn: ({
      formValue: { provider, dbName, k8sNamespace, ...rest },
    }: CreateInstanceHookArgType) => {
      const spec = {
        ...rest,
      };
      return createInstanceFn(
        'main',
        dbName,
        provider,
        k8sNamespace || '',
        spec
      );
    },
    ...options,
  });

export default useCreateInstance;
