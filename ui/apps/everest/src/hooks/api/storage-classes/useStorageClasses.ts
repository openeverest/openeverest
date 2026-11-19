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

import { useQuery } from '@tanstack/react-query';
import { getStorageClassesFn } from 'api/storageClassesApi';
import type {
  StorageClass,
  StorageClassList,
} from 'shared-types/kubernetes.types';
import { sortDefaultStorageClassesFirst } from './storage-classes.utils';

const getStorageClassesQueryKey = (clusterName: string) =>
  ['storage-classes', clusterName] as const;

const selectStorageClasses = ({ items }: StorageClassList) =>
  sortDefaultStorageClassesFirst(items);

// Not gated on useRBACPermissions: it checks the `<cluster>/*` object, which a
// role granted only specific classes does not match, while the API already
// filters the list down to the classes the user may read.
export const useStorageClassesList = (clusterName: string) =>
  useQuery<StorageClassList, unknown, StorageClass[]>({
    queryKey: getStorageClassesQueryKey(clusterName),
    queryFn: () => getStorageClassesFn(clusterName),
    select: selectStorageClasses,
  });
