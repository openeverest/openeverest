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
import { sortDefaultStorageClassesFirst } from './storage-classes.utils';

const makeStorageClass = (
  name: string,
  { isDefault = false, createdAt = '2026-01-01T00:00:00Z' } = {}
): StorageClass => ({
  metadata: {
    name,
    creationTimestamp: createdAt,
    ...(isDefault
      ? {
          annotations: {
            'storageclass.kubernetes.io/is-default-class': 'true',
          },
        }
      : {}),
  },
  provisioner: 'rancher.io/local-path',
});

const names = (storageClasses: StorageClass[]) =>
  storageClasses.map((sc) => sc.metadata?.name);

describe('sortDefaultStorageClassesFirst', () => {
  it('moves the default class first and keeps the others in API order', () => {
    const sorted = sortDefaultStorageClassesFirst([
      makeStorageClass('gp2'),
      makeStorageClass('io1'),
      makeStorageClass('standard', { isDefault: true }),
      makeStorageClass('fast'),
    ]);

    expect(names(sorted)).toEqual(['standard', 'gp2', 'io1', 'fast']);
  });

  it('puts the newest default first when several classes are default', () => {
    const sorted = sortDefaultStorageClassesFirst([
      makeStorageClass('old-default', {
        isDefault: true,
        createdAt: '2025-01-01T00:00:00Z',
      }),
      makeStorageClass('plain'),
      makeStorageClass('new-default', {
        isDefault: true,
        createdAt: '2026-06-01T00:00:00Z',
      }),
    ]);

    expect(names(sorted)).toEqual(['new-default', 'old-default', 'plain']);
  });

  it('ignores a default annotation that is not "true"', () => {
    const notDefault: StorageClass = {
      metadata: {
        name: 'false-default',
        annotations: { 'storageclass.kubernetes.io/is-default-class': 'false' },
      },
      provisioner: 'rancher.io/local-path',
    };

    const sorted = sortDefaultStorageClassesFirst([
      notDefault,
      makeStorageClass('standard', { isDefault: true }),
    ]);

    expect(names(sorted)).toEqual(['standard', 'false-default']);
  });
});
