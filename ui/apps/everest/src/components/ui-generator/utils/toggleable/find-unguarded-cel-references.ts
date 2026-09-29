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

import type {
  Component,
  ComponentGroup,
  Section,
} from 'components/ui-generator/ui-generator.types';
import { getToggleableMeta } from './toggleable';

// A `spec.` path not preceded by an identifier or a dot, so `original.spec.x` is skipped.
const SPEC_REF = /(^|[^\w.])(spec(?:\.\w+)+)/g;
const HAS_CALL = /has\(\s*(spec(?:\.\w+)+)\s*\)/g;

export interface UnguardedCelReference {
  // Dotted key paths: section key followed by group/field keys.
  field: string;
  group: string;
  path: string;
}

interface ActiveToggleable {
  key: string;
  paths: string[];
}

interface CelField {
  key: string;
  enclosing?: string;
  exprs: string[];
}

const getCelExpressions = (item: Component | ComponentGroup): string[] => {
  const validation = 'validation' in item ? item.validation : undefined;
  return validation
    ? [validation, ...Object.values(validation.modes ?? {})]
        .flatMap((branch) => branch?.celExpressions ?? [])
        .map(({ celExpr }) => celExpr)
        .filter(Boolean)
    : [];
};

const collect = (
  components: Record<string, Component | ComponentGroup>,
  parentKey: string,
  enclosing: string | undefined,
  toggleables: ActiveToggleable[],
  celFields: CelField[]
) => {
  Object.entries(components).forEach(([key, item]) => {
    const keyPath = `${parentKey}.${key}`;
    if (item.uiType === 'group' || item.uiType === 'hidden') {
      if (!('components' in item) || !item.components) return;
      const meta = getToggleableMeta(item);
      if (meta) toggleables.push({ key: keyPath, paths: meta.childPaths });
      collect(
        item.components,
        keyPath,
        meta ? keyPath : enclosing,
        toggleables,
        celFields
      );
      return;
    }
    const exprs = getCelExpressions(item);
    if (exprs.length > 0) celFields.push({ key: keyPath, enclosing, exprs });
  });
};

const findUnguardedPath = (
  exprs: string[],
  groupPaths: string[]
): string | undefined => {
  for (const expr of exprs) {
    const guarded = new Set(Array.from(expr.matchAll(HAS_CALL), (m) => m[1]));
    for (const [, , ref] of expr.matchAll(SPEC_REF)) {
      const path = groupPaths.find(
        (p) => (ref === p || ref.startsWith(`${p}.`)) && !guarded.has(p)
      );
      if (path) return path;
    }
  }
  return undefined;
};

// A switched-off group's fields are absent for CEL outside it, so an unguarded
// reference makes that rule fail. Expects preprocessed sections.
export const findUnguardedCelReferences = (
  sections: Record<string, Section>
): UnguardedCelReference[] => {
  const toggleables: ActiveToggleable[] = [];
  const celFields: CelField[] = [];
  Object.entries(sections).forEach(([sectionKey, section]) => {
    if (section?.components) {
      collect(
        section.components,
        sectionKey,
        undefined,
        toggleables,
        celFields
      );
    }
  });

  return celFields.flatMap((field) =>
    toggleables
      .filter((group) => group.key !== field.enclosing)
      .flatMap((group) => {
        const path = findUnguardedPath(field.exprs, group.paths);
        return path ? [{ field: field.key, group: group.key, path }] : [];
      })
  );
};
