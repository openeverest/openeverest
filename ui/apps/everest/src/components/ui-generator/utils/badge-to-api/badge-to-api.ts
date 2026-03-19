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

import {
  Component,
  ComponentGroup,
  TopologyUISchemas,
} from '../../ui-generator.types';

export type BadgeMapping = {
  path: string;
  badge: string;
};

const normalizeComponentPaths = (path: string | string[]): string[] => {
  if (Array.isArray(path)) {
    return path.filter((p): p is string => typeof p === 'string' && !!p);
  }

  return typeof path === 'string' && path ? [path] : [];
};

export const extractBadgeMappings = (
  schema: TopologyUISchemas,
  selectedTopology: string
): BadgeMapping[] => {
  const badgeMappings: BadgeMapping[] = [];
  const topology = schema[selectedTopology];

  if (!topology || !topology.sections) {
    return badgeMappings;
  }

  const processComponents = (components: {
    [key: string]: Component | ComponentGroup;
  }): void => {
    Object.values(components).forEach((item) => {
      if (item.uiType === 'group' && 'components' in item) {
        // Recursively process group components
        processComponents((item as ComponentGroup).components);
      } else {
        // Handle regular component
        const component = item as Component;
        if (
          'path' in component &&
          component.path &&
          component.fieldParams?.badge &&
          component.fieldParams?.badgeToApi
        ) {
          normalizeComponentPaths(component.path).forEach((path) => {
            badgeMappings.push({
              path,
              badge: component.fieldParams.badge!,
            });
          });
        }
      }
    });
  };

  Object.values(topology.sections).forEach((section) => {
    if (section?.components) {
      processComponents(section.components);
    }
  });

  return badgeMappings;
};

/**
 * Applies badge suffixes to form data based on badge mappings
 * Converts dot-notation paths to nested object access and appends badges
 */
export const applyBadgesToFormData = (
  formData: Record<string, unknown>,
  badgeMappings: BadgeMapping[]
): Record<string, unknown> => {
  if (badgeMappings.length === 0) {
    return formData;
  }

  // Deep clone to avoid mutating original data
  const result = JSON.parse(JSON.stringify(formData)) as Record<
    string,
    unknown
  >;

  badgeMappings.forEach(({ path, badge }) => {
    if (!path || typeof path !== 'string') {
      return;
    }

    const pathParts = path.split('.');
    let current: Record<string, unknown> = result;

    // Navigate to the parent object
    for (let i = 0; i < pathParts.length - 1; i++) {
      const part = pathParts[i];
      if (!current[part]) {
        current[part] = {};
      }
      current = current[part] as Record<string, unknown>;
    }

    // Get the final key and apply badge
    const finalKey = pathParts[pathParts.length - 1];
    const value = current[finalKey];

    if (value !== undefined && value !== null && value !== '') {
      // Convert to string, trim, and append badge
      current[finalKey] = `${String(value).trim()}${badge}`;
    }
  });

  return result;
};
