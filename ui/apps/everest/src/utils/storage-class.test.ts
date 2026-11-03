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
import { getDefaultStorageClassName } from './storage-class';

const storageClass = (name: string, isDefault = false): StorageClass => ({
  allowVolumeExpansion: true,
  metadata: {
    name,
    annotations: isDefault
      ? { 'storageclass.kubernetes.io/is-default-class': 'true' }
      : undefined,
  },
});

describe('getDefaultStorageClassName', () => {
  it('returns the first class when it is annotated as default', () => {
    expect(
      getDefaultStorageClassName({
        clusterType: 'generic',
        storageClassNames: ['gp3', 'gp2'],
        storageClasses: [storageClass('gp2'), storageClass('gp3', true)],
      })
    ).toBe('gp3');
  });

  it('returns undefined when the cluster has no default class', () => {
    expect(
      getDefaultStorageClassName({
        clusterType: 'generic',
        storageClassNames: ['gp2', 'gp3'],
        storageClasses: [storageClass('gp2'), storageClass('gp3')],
      })
    ).toBeUndefined();
  });

  it('returns undefined before cluster info is loaded', () => {
    expect(getDefaultStorageClassName(undefined)).toBeUndefined();
  });
});
