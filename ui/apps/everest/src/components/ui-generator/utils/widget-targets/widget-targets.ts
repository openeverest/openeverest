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
  Section,
  WidgetComponent,
  WidgetTarget,
  isWidgetComponent,
} from '../../ui-generator.types';
import { getByPath } from '../object-path';

export const getWidgetTargets = (
  item: Component | ComponentGroup
): WidgetTarget[] =>
  !('components' in item) && isWidgetComponent(item)
    ? (item._widgetTargets ?? [])
    : [];

export const getWidgetTargetPaths = (
  item: Component | ComponentGroup
): string[] => getWidgetTargets(item).map(({ path }) => path);

// A marker widget's value: target key → value at its path in `source` (an
// instance in the overview, form values in the wizard).
export const readWidgetTargetValues = (
  targets: WidgetTarget[],
  source: Record<string, unknown>
): Record<string, unknown> =>
  Object.fromEntries(
    targets.map(({ key, path }) => [key, getByPath(source, path)])
  );

export type ResolveWidgetTargets = (
  widget: WidgetComponent
) => WidgetTarget[] | undefined;

const withTargetsInComponents = (
  components: Record<string, Component | ComponentGroup>,
  resolve: ResolveWidgetTargets
): Record<string, Component | ComponentGroup> =>
  Object.fromEntries(
    Object.entries(components).map(([key, item]) => {
      if ('components' in item) {
        return [
          key,
          {
            ...item,
            components: withTargetsInComponents(item.components, resolve),
          },
        ];
      }
      const targets = isWidgetComponent(item) ? resolve(item) : undefined;
      return [key, targets ? { ...item, _widgetTargets: targets } : item];
    })
  );

// Runs before toggleable resolution, which must already see the paths a marker
// widget writes.
export const withWidgetTargets = (
  sections: Record<string, Section>,
  resolve: ResolveWidgetTargets
): Record<string, Section> =>
  Object.fromEntries(
    Object.entries(sections).map(([key, section]) => [
      key,
      section?.components
        ? {
            ...section,
            components: withTargetsInComponents(section.components, resolve),
          }
        : section,
    ])
  );
