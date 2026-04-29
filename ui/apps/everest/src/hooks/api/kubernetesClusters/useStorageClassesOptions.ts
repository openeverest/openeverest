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

import type { ProviderOptions } from 'components/ui-generator/api-providers/types';
import { useKubernetesClusterInfo } from './useKubernetesClusterInfo';

// TODO: useKubernetesClusterInfo currently requires queryKey as a parameter
// (legacy API from react-query v3 migration). Every caller invents its own key,
// which prevents proper React Query cache sharing. When refactoring this hook
// check section-edit-modal still work correctly with a shared canonical key.

const STORAGE_CLASSES_QUERY_KEY = ['kubernetesClusterInfo'];

export const useStorageClassesOptions = (): ProviderOptions => {
  const { data, isLoading, error } = useKubernetesClusterInfo(
    STORAGE_CLASSES_QUERY_KEY
  );

  const names = data?.storageClassNames ?? [];
  const options = names.map((name) => ({ label: name, value: name }));

  return {
    options,
    isLoading,
    error,
    isEmpty: !isLoading && options.length === 0,
    rawData: data?.storageClassNames,
  };
};
