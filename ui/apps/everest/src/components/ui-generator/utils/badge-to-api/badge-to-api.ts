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

import type { Section, TopologyUISchemas } from '../../ui-generator.types';
import { walkLeafComponents, walkTopologyComponents } from '../schema-walker';
import { getComponentTargetPaths } from '../preprocess/normalized-component';
import { getByPath, setByPath, deepClone } from '../object-path';

export type BadgeMapping = {
  path: string;
  badge: string;
};

export const extractBadgeMappings = (
  schema: TopologyUISchemas,
  selectedTopology: string
): BadgeMapping[] => {
  const badgeMappings: BadgeMapping[] = [];

  walkTopologyComponents(schema, selectedTopology, ({ component }) => {
    if (component.fieldParams?.badge && component.fieldParams?.badgeToApi) {
      getComponentTargetPaths(component).forEach((path) => {
        badgeMappings.push({
          path,
          badge: component.fieldParams.badge!,
        });
      });
    }
  });

  return badgeMappings;
};

export const applyBadgesToFormData = (
  formData: Record<string, unknown>,
  badgeMappings: BadgeMapping[]
): Record<string, unknown> => {
  if (badgeMappings.length === 0) {
    return formData;
  }

  const result = deepClone(formData);

  badgeMappings.forEach(({ path, badge }) => {
    const value = getByPath(result, path);

    if (value !== undefined && value !== null && value !== '') {
      setByPath(result, path, `${String(value).trim()}${badge}`);
    }
  });

  return result;
};

export const stripBadgeFromValue = (
  value: unknown,
  badge?: string
): unknown => {
  if (typeof value !== 'string' || !badge) {
    return value;
  }

  const trimmedValue = value.trim();
  if (!trimmedValue.endsWith(badge)) {
    return value;
  }

  return trimmedValue.slice(0, -badge.length).trimEnd();
};

export const extractBadgeMappingsFromSections = (
  sections: Record<string, Section>
): BadgeMapping[] => {
  const badgeMappings: BadgeMapping[] = [];

  for (const section of Object.values(sections)) {
    if (!section?.components) continue;
    walkLeafComponents(section.components, ({ component }) => {
      if (component.fieldParams?.badge && component.fieldParams?.badgeToApi) {
        getComponentTargetPaths(component).forEach((path) => {
          badgeMappings.push({ path, badge: component.fieldParams.badge! });
        });
      }
    });
  }

  return badgeMappings;
};

/*
Strips badge suffixes from fields in a nested data object and coerces the
resulting strings to numbers so CEL numeric comparisons work correctly.
e.g. { spec: { engine: { storage: { size: "25Gi" } } } } → { …size: 25 }
*/
export const stripBadgesFromData = (
  data: Record<string, unknown>,
  badgeMappings: BadgeMapping[]
): Record<string, unknown> => {
  if (badgeMappings.length === 0) return data;

  const result = deepClone(data);

  badgeMappings.forEach(({ path, badge }) => {
    const value = getByPath(result, path);
    if (value === undefined) return;

    const stripped = stripBadgeFromValue(value, badge);
    if (typeof stripped === 'string' && stripped !== '') {
      const asNumber = Number(stripped.trim());
      setByPath(result, path, Number.isNaN(asNumber) ? stripped : asNumber);
    } else {
      setByPath(result, path, stripped);
    }
  });

  return result;
};
