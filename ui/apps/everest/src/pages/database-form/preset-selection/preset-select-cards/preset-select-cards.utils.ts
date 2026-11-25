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

import { CardPickerItem, flattenToSearchText } from '@percona/ui-lib';
import {
  Component,
  FieldType,
  Section,
  Topology,
} from 'components/ui-generator/ui-generator.types';
import { orderComponents } from 'components/ui-generator/utils/component-renderer/order-components';
import { getByPath } from 'components/ui-generator/utils/object-path/object-path';
import { getComponentTargetPaths } from 'components/ui-generator/utils/preprocess/normalized-component';
import { formatQuantity } from 'components/ui-generator/utils/quantity/quantity';
import { isKubernetesMemoryUnit } from 'utils/k8ResourceParser';
import { PRESET_BADGE_COLOR_BY_SCOPE } from '../preset-selection.constants';
import { DroppedPreset, SelectablePreset } from '../preset-selection.types';
import { Messages } from './preset-select-cards.messages';

const RESOURCE_PATH =
  /^spec\.components\.([^.]+)\.(replicas|resources\..+|storage\.size)$/;

type ComponentResourceDetails = Map<string, Map<string, string>>;
type PresetComponents = NonNullable<SelectablePreset['spec']>['components'];

const toText = (value: number | string | undefined): string | undefined =>
  value === undefined || value === null ? undefined : String(value);

const getMeta = (preset: SelectablePreset): string =>
  [preset.spec?.topology?.type, preset.spec?.version]
    .filter(Boolean)
    .join(' · ');

const addFallbackResources = (
  components: PresetComponents,
  details: ComponentResourceDetails
): void => {
  for (const [name, { replicas, resources, storage }] of Object.entries(
    components ?? {}
  )) {
    const cpu = toText(resources?.limits?.cpu);
    const memory = toText(resources?.limits?.memory);
    const disk = toText(storage?.size);
    const fallback = {
      replicas:
        typeof replicas === 'number'
          ? Messages.summary.nodes(replicas)
          : undefined,
      'resources.limits.cpu': cpu && Messages.summary.cpu(cpu),
      'resources.limits.memory': memory && Messages.summary.memory(memory),
      'storage.size': disk && Messages.summary.disk(disk),
    };
    const fields = details.get(name) ?? new Map<string, string>();
    for (const [path, summary] of Object.entries(fallback)) {
      if (summary && !fields.has(path)) fields.set(path, summary);
    }
    if (fields.size > 0) details.set(name, fields);
  }
};

const formatResourceValue = (component: Component, value: string): string => {
  if (component.uiType === FieldType.Quantity) {
    const { unit, unitLabel } = component.fieldParams;
    return formatQuantity(value, unit, unitLabel) ?? value;
  }

  const { badge, badgeToApi } = component.fieldParams ?? {};
  if (badgeToApi && badge && isKubernetesMemoryUnit(badge)) {
    return formatQuantity(value, badge) ?? value;
  }

  return value;
};

const collectResourceDetails = (
  {
    components,
    componentsOrder,
  }: Pick<Section, 'components' | 'componentsOrder'>,
  instance: { spec: SelectablePreset['spec'] },
  details: ComponentResourceDetails
): void => {
  for (const [key, component] of orderComponents(components, componentsOrder)) {
    if (component.uiType === FieldType.Hidden) continue;
    if ('components' in component) {
      collectResourceDetails(component, instance, details);
      continue;
    }

    for (const path of getComponentTargetPaths(component)) {
      const match = RESOURCE_PATH.exec(path);
      if (!match) continue;

      const [, name, resourcePath] = match;
      const fields = details.get(name) ?? new Map<string, string>();
      if (fields.has(resourcePath)) continue;

      const value = getByPath(instance, path);
      if (
        typeof value !== 'number' &&
        (typeof value !== 'string' || !value.trim())
      ) {
        continue;
      }

      const label = component.fieldParams?.label || key;
      const display = formatResourceValue(component, String(value));
      fields.set(resourcePath, Messages.summary.field(label, display));
      details.set(name, fields);
    }
  }
};

const getResources = (
  preset: SelectablePreset,
  topology?: Topology
): string => {
  const details: ComponentResourceDetails = new Map();
  const instance = { spec: preset.spec };
  const sections = topology?.sections ?? {};
  const sectionKeys = new Set([
    ...(topology?.sectionsOrder ?? []),
    ...Object.keys(sections),
  ]);
  for (const key of sectionKeys) {
    const section = sections[key];
    if (section) collectResourceDetails(section, instance, details);
  }

  const components = preset.spec?.components ?? {};
  addFallbackResources(components, details);
  const hasMultipleComponents = Object.keys(components).length > 1;
  return Array.from(details, ([name, fields]) => {
    const summary = Array.from(fields.values()).join(' · ');
    return hasMultipleComponents
      ? Messages.summary.component(name, summary)
      : summary;
  }).join(' | ');
};

export const presetToCardItem = (
  preset: SelectablePreset,
  topology?: Topology
): CardPickerItem => {
  const subtitle = getMeta(preset);
  const detail = getResources(preset, topology);
  return {
    id: preset.id,
    title: preset.name,
    badge: {
      label: Messages.scope[preset.scope],
      color: PRESET_BADGE_COLOR_BY_SCOPE[preset.scope],
    },
    subtitle,
    detail,
    // Include the full spec so values like storage class stay searchable.
    searchText: `${preset.name} ${Messages.scope[preset.scope]} ${detail} ${flattenToSearchText(preset.spec)}`,
  };
};

// A same-named preset may exist in the new namespace, so a namespace switch
// names the namespace the dropped one belongs to.
export const getDroppedPresetMessage = ({
  name,
  wizardNamespace,
  ownerNamespace,
}: DroppedPreset): string =>
  ownerNamespace && ownerNamespace !== wizardNamespace
    ? Messages.presetFromOtherNamespace(name, ownerNamespace, wizardNamespace)
    : Messages.presetUnavailable(name);
