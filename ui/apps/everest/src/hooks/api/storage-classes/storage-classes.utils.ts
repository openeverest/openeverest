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

import type { StorageClass } from 'shared-types/kubernetes.types';

const DEFAULT_STORAGE_CLASS_ANNOTATION =
  'storageclass.kubernetes.io/is-default-class';

const isDefaultStorageClass = (storageClass: StorageClass) =>
  storageClass.metadata?.annotations?.[DEFAULT_STORAGE_CLASS_ANNOTATION] ===
  'true';

const createdAt = (storageClass: StorageClass) =>
  Date.parse(storageClass.metadata?.creationTimestamp ?? '') || 0;

// Data-source fields preselect the first option, so the default class goes
// first. With several defaults Kubernetes uses the newest one, so newer
// defaults go first. The API does not order the list, so the UI preserves the
// default-first behavior when selecting an initial option.
export const sortDefaultStorageClassesFirst = (
  storageClasses: StorageClass[]
): StorageClass[] =>
  [...storageClasses].sort((a, b) => {
    const aDefault = isDefaultStorageClass(a);
    const bDefault = isDefaultStorageClass(b);

    if (aDefault !== bDefault) {
      return aDefault ? -1 : 1;
    }

    return aDefault ? createdAt(b) - createdAt(a) : 0;
  });
