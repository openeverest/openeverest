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
  NormalizedPathMeta,
} from '../../ui-generator.types';

/**
 * Normalized path metadata computed during preprocessing.
 *
 * Schema components declare their binding via `path: string | string[]`.
 * - Single path  (`"spec.replicas"`)  → read from and write to the same location.
 * - Array of paths (`["spec.engine.version", "spec.proxy.version"]`)
 *   → the first entry is the *source* (the form field reads its value from here),
 *     and every entry is a *target* (the value is written back to all paths on submit).
 *
 * Preprocessing normalises this into `NormalizedPathMeta` so downstream code
 * never has to branch on the original shape.
 */

const toNormalizedPathMeta = (
  path: string | string[] | undefined
): NormalizedPathMeta => {
  if (typeof path === 'string' && path) {
    return {
      sourcePath: path,
      targetPaths: [path],
    };
  }

  if (Array.isArray(path)) {
    const normalized = Array.from(
      new Set(path.filter((p): p is string => typeof p === 'string' && !!p))
    );

    return {
      sourcePath: normalized[0],
      targetPaths: normalized,
    };
  }

  return {
    sourcePath: undefined,
    targetPaths: [],
  };
};

export const withNormalizedPathMeta = (component: Component): Component => {
  if (component._normalized) {
    return component;
  }

  return {
    ...component,
    _normalized: toNormalizedPathMeta(
      'path' in component ? component.path : undefined
    ),
  };
};

export const getComponentSourcePath = (
  item: Component | ComponentGroup
): string | undefined => {
  if (
    !('uiType' in item) ||
    item.uiType === 'group' ||
    item.uiType === 'hidden'
  ) {
    return undefined;
  }

  const component = item as Component;

  if (component._normalized?.sourcePath) {
    return component._normalized.sourcePath;
  }

  return toNormalizedPathMeta('path' in component ? component.path : undefined)
    .sourcePath;
};

export const getComponentTargetPaths = (
  item: Component | ComponentGroup
): string[] => {
  if (
    !('uiType' in item) ||
    item.uiType === 'group' ||
    item.uiType === 'hidden'
  ) {
    return [];
  }

  const component = item as Component;

  if (component._normalized?.targetPaths) {
    return component._normalized.targetPaths;
  }

  return toNormalizedPathMeta('path' in component ? component.path : undefined)
    .targetPaths;
};
