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

import { buildSectionFieldMap } from './build-section-field-map';
import { FieldType, Section } from '../../ui-generator.types';

const makeSections = (paths: string[]): { [key: string]: Section } => ({
  resources: {
    label: 'Resources',
    components: Object.fromEntries(
      paths.map((p, i) => [
        `field${i}`,
        {
          uiType: FieldType.Number,
          path: p,
          fieldParams: { label: `Field ${i}` },
        },
      ])
    ),
  },
});

describe('buildSectionFieldMap', () => {
  it('maps the leaf path to the correct section key', () => {
    const sections = makeSections(['spec.components.configServer.replicas']);
    const map = buildSectionFieldMap(sections, ['resources']);

    expect(map['spec.components.configServer.replicas']).toBe('resources');
  });

  it('registers ALL intermediate path prefixes for robust error-path lookup', () => {
    // when topology switches, zod may report errors at intermediate paths (spec.components.configServer)
    // instead of the (spec.components.configServer.replicas) because the nested object is
    // undefined. All intermediate prefixes must map to the same step so that
    // stepsWithErrors always highlights the correct step.
    const sections = makeSections(['spec.components.configServer.replicas']);
    const map = buildSectionFieldMap(sections, ['resources']);

    expect(map['spec']).toBe('resources');
    expect(map['spec.components']).toBe('resources');
    expect(map['spec.components.configServer']).toBe('resources');
    expect(map['spec.components.configServer.replicas']).toBe('resources');
  });

  it('does not override an intermediate prefix already registered by another field in a different section', () => {
    // "spec" is first registered by "spec.databaseVersion" in section dbVersion.
    // Later fields in resources have paths inside "spec.components.*", but since
    // "spec" is already in the map it must NOT be overwritten.
    const sections: { [key: string]: Section } = {
      dbVersion: {
        label: 'DB Version',
        components: {
          ver: {
            uiType: FieldType.Text,
            path: 'spec.databaseVersion',
            fieldParams: { label: 'Version' },
          },
        },
      },
      resources: {
        label: 'Resources',
        components: {
          replicas: {
            uiType: FieldType.Number,
            path: 'spec.components.configServer.replicas',
            fieldParams: { label: 'Replicas' },
          },
        },
      },
    };

    const map = buildSectionFieldMap(sections, ['dbVersion', 'resources']);

    expect(map['spec.databaseVersion']).toBe('dbVersion');
    // "spec" was first registered when processing "spec.databaseVersion"
    expect(map['spec']).toBe('dbVersion');
    expect(map['spec.components.configServer.replicas']).toBe('resources');
    // intermediate paths for the replicas field that weren't yet registered:
    expect(map['spec.components']).toBe('resources');
    expect(map['spec.components.configServer']).toBe('resources');
  });

  it('handles top-level (no-dot) paths', () => {
    const sections = makeSections(['dbName']);
    const map = buildSectionFieldMap(sections, ['resources']);

    expect(map['dbName']).toBe('resources');
    // A single-segment path has no intermediate prefixes to register
    expect(Object.keys(map)).toEqual(['dbName']);
  });

  it('handles multiple fields in the same section', () => {
    const sections = makeSections([
      'spec.components.configServer.replicas',
      'spec.components.proxy.replicas',
      'spec.sharding.enabled',
    ]);
    const map = buildSectionFieldMap(sections, ['resources']);

    expect(map['spec.components.configServer.replicas']).toBe('resources');
    expect(map['spec.components.proxy.replicas']).toBe('resources');
    expect(map['spec.sharding.enabled']).toBe('resources');

    expect(map['spec']).toBe('resources');
    expect(map['spec.components']).toBe('resources');
    expect(map['spec.components.configServer']).toBe('resources');
    expect(map['spec.components.proxy']).toBe('resources');
    expect(map['spec.sharding']).toBe('resources');
  });

  it('maps all target paths for multipath fields', () => {
    const sections: { [key: string]: Section } = {
      resources: {
        label: 'Resources',
        components: {
          engineVersion: {
            uiType: FieldType.Text,
            path: ['spec.engine.version', 'spec.proxy.version'],
            fieldParams: { label: 'Version' },
          },
        },
      },
    };

    const map = buildSectionFieldMap(sections, ['resources']);

    expect(map['spec.engine.version']).toBe('resources');
    expect(map['spec.proxy.version']).toBe('resources');
    // Source field id for multipath is the first path entry, not generated g-* id.
    expect(map['g-engineVersion']).toBeUndefined();
  });
});
