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

import type { KubernetesClusterInfo } from 'shared-types/kubernetes.types';

const DEFAULT_STORAGE_CLASS_ANNOTATION =
  'storageclass.kubernetes.io/is-default-class';

// The server lists default storage classes first (newest first), so the
// cluster default is the first name — but only if it actually carries the
// default annotation; a cluster may have no default at all.
export const getDefaultStorageClassName = (
  clusterInfo: KubernetesClusterInfo | undefined
): string | undefined => {
  const first = clusterInfo?.storageClassNames[0];
  if (!first) return undefined;

  const isDefault = (clusterInfo.storageClasses ?? []).some(
    (storageClass) =>
      storageClass.metadata.name === first &&
      storageClass.metadata.annotations?.[DEFAULT_STORAGE_CLASS_ANNOTATION] ===
        'true'
  );

  return isDefault ? first : undefined;
};
