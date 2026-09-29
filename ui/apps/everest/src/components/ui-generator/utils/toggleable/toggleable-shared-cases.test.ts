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

import { parse } from 'yaml';
import { describe, expect, it } from 'vitest';
import {
  Component,
  ComponentGroup,
  GroupType,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { getToggleableMeta } from './toggleable';
import { findUnguardedCelReferences } from './find-unguarded-cel-references';
// Also run by the Go validator (pkg/uischema), so both apply the same rules.
import sharedCasesYaml from '../../../../../../../../pkg/uischema/testdata/toggleable.yaml?raw';

interface SharedIssue {
  rule: string;
  topology: string;
  group: string;
  field?: string;
}

interface SharedCase {
  name: string;
  uiSchema: TopologyUISchemas;
  issues: SharedIssue[];
}

const { cases }: { cases: SharedCase[] } = parse(sharedCasesYaml);

const CEL_RULE = 'toggleable-cel-unguarded';

// The UI only exposes whether a group degraded, not which rule did it.
const collectDegraded = (
  declared: Record<string, Component | ComponentGroup>,
  processed: Record<string, Component | ComponentGroup>,
  parentKey: string,
  degraded: string[]
) => {
  Object.entries(declared).forEach(([key, item]) => {
    const result = processed[key];
    if (!('components' in item) || !('components' in result)) return;
    const keyPath = `${parentKey}.${key}`;
    if (
      item.uiType === 'group' &&
      item.groupType === GroupType.Toggleable &&
      !getToggleableMeta(result)
    ) {
      degraded.push(keyPath);
    }
    collectDegraded(item.components, result.components, keyPath, degraded);
  });
};

describe('toggleable rules shared with the Go validator', () => {
  it.each(cases)('$name', ({ uiSchema, issues }) => {
    const processed = preprocessSchema(uiSchema);
    const degraded: string[] = [];
    const unguarded: string[] = [];

    Object.entries(uiSchema).forEach(([topology, { sections }]) => {
      const processedSections = processed[topology].sections;
      Object.entries(sections).forEach(([sectionKey, section]) => {
        const groups: string[] = [];
        collectDegraded(
          section.components,
          processedSections[sectionKey].components,
          sectionKey,
          groups
        );
        groups.forEach((group) => degraded.push(`${topology}:${group}`));
      });
      findUnguardedCelReferences(processedSections).forEach(
        ({ group, field }) => unguarded.push(`${topology}:${group}:${field}`)
      );
    });

    expect(degraded.sort()).toEqual(
      issues
        .filter(({ rule }) => rule !== CEL_RULE)
        .map(({ topology, group }) => `${topology}:${group}`)
        .sort()
    );
    expect(unguarded.sort()).toEqual(
      issues
        .filter(({ rule }) => rule === CEL_RULE)
        .map(({ topology, group, field }) => `${topology}:${group}:${field}`)
        .sort()
    );
  });
});
