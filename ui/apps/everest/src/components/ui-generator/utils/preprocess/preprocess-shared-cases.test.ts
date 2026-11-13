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
  FieldType,
  GroupType,
  Section,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { walkLeafComponents } from '../schema-walker';
import { getToggleableMeta } from '../toggleable/toggleable';
import { findUnguardedCelReferences } from '../toggleable/find-unguarded-cel-references';
import { preprocessSchema } from './preprocess-schema';

// The Go validator (provider-runtime/uischema) runs every file too, so both
// apply the same rules.
const sharedCaseFiles = import.meta.glob<string>(
  '../../../../../../../../provider-runtime/uischema/testdata/*.yaml',
  { query: '?raw', import: 'default', eager: true }
);

interface SharedIssue {
  rule: string;
  topology: string;
  group?: string;
  field?: string;
}

interface SharedCase {
  file: string;
  name: string;
  uiSchema: TopologyUISchemas;
  issues: SharedIssue[];
}

const cases: SharedCase[] = Object.entries(sharedCaseFiles).flatMap(
  ([path, yaml]) => {
    const { cases: fileCases }: { cases: Omit<SharedCase, 'file'>[] } =
      parse(yaml);
    const file = path.slice(path.lastIndexOf('/') + 1);
    return fileCases.map((sharedCase) => ({ ...sharedCase, file }));
  }
);

const CEL_RULE = 'toggleable-cel-unguarded';

// The UI only shows the outcome of a rule, not which rule it was.
const toUiOutcome = ({ rule, topology, group, field }: SharedIssue): string => {
  if (rule === CEL_RULE) return `cel ${topology}:${group}:${field}`;
  if (rule.startsWith('toggleable-')) return `bordered ${topology}:${group}`;
  if (rule.startsWith('array-')) return `dropped ${topology}:${field}`;
  throw new Error(`No UI outcome to compare rule ${rule} with; add it here.`);
};

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

const collectFieldKeys = (
  sections: Record<string, Section>,
  isIncluded: (component: Component) => boolean = () => true
): string[] => {
  const keys: string[] = [];
  Object.entries(sections).forEach(([sectionKey, { components }]) =>
    walkLeafComponents(
      components,
      ({ component, generatedName }) => {
        if (isIncluded(component)) keys.push(generatedName);
      },
      sectionKey
    )
  );
  return keys;
};

describe('uiSchema rules shared with the Go validator', () => {
  it.each(cases)('$file: $name', ({ uiSchema, issues }) => {
    const processed = preprocessSchema(uiSchema);
    const outcomes: string[] = [];

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
        groups.forEach((group) =>
          outcomes.push(`bordered ${topology}:${group}`)
        );
      });
      findUnguardedCelReferences(processedSections).forEach(
        ({ group, field }) => outcomes.push(`cel ${topology}:${group}:${field}`)
      );
      const rendered = new Set(collectFieldKeys(processedSections));
      collectFieldKeys(sections, ({ uiType }) => uiType === FieldType.Array)
        .filter((key) => !rendered.has(key))
        .forEach((key) => outcomes.push(`dropped ${topology}:${key}`));
    });

    expect(outcomes.sort()).toEqual(issues.map(toUiOutcome).sort());
  });
});
