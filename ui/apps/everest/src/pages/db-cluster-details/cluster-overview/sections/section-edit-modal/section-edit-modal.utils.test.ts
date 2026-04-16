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

import { applyRuntimeOverrides } from './section-edit-modal.utils';
import type { Instance } from 'shared-types/api.types';
import type { KubernetesClusterInfo } from 'shared-types/kubernetes.types';
import {
  FieldType,
  type Section,
} from 'components/ui-generator/ui-generator.types';

const STORAGE_SIZE_PATH = 'spec.components.engine.storage.size';

const makeInstance = (storageClass: string): Instance =>
  ({
    spec: {
      components: {
        engine: {
          storage: { storageClass },
        },
      },
    },
  }) as unknown as Instance;

const makeClusterInfo = (
  name: string,
  allowVolumeExpansion: boolean
): KubernetesClusterInfo => ({
  clusterType: 'k8s',
  storageClassNames: [name],
  storageClasses: [{ metadata: { name }, allowVolumeExpansion }],
});

const makeSections = (): Record<string, Section> => ({
  resources: {
    label: 'Resources',
    components: {
      diskSize: {
        uiType: FieldType.Number,
        path: STORAGE_SIZE_PATH,
        fieldParams: { label: 'Disk size' },
      },
    },
  },
});

describe('applyRuntimeOverrides', () => {
  it('disables storage size field when allowVolumeExpansion is false', () => {
    const result = applyRuntimeOverrides(
      makeSections(),
      makeInstance('standard'),
      makeClusterInfo('standard', false)
    );

    const component = result.resources.components.diskSize;
    expect('fieldParams' in component && component.fieldParams).toEqual(
      expect.objectContaining({
        disabled: true,
        tooltip: expect.stringContaining("can't change the disk size"),
      })
    );
  });

  it('does not modify sections when allowVolumeExpansion is true', () => {
    const sections = makeSections();
    const result = applyRuntimeOverrides(
      sections,
      makeInstance('standard'),
      makeClusterInfo('standard', true)
    );

    // Should return the same reference (no enrichment needed)
    expect(result).toBe(sections);
  });

  it('does not modify sections when storage class is not found in cluster info', () => {
    const sections = makeSections();
    const result = applyRuntimeOverrides(
      sections,
      makeInstance('unknown-class'),
      makeClusterInfo('standard', false)
    );

    expect(result).toBe(sections);
  });

  it('does not modify sections when cluster info is undefined', () => {
    const sections = makeSections();
    const result = applyRuntimeOverrides(sections, makeInstance('standard'));

    expect(result).toBe(sections);
  });

  it('does not modify sections when instance has no storage class', () => {
    const sections = makeSections();
    const result = applyRuntimeOverrides(
      sections,
      { spec: {} } as unknown as Instance,
      makeClusterInfo('standard', false)
    );

    expect(result).toBe(sections);
  });
});
