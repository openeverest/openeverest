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
  FieldType,
  FormMode,
  GroupType,
  SelectFieldParams,
  TopologyUISchemas,
  isMapComponent,
} from '../../ui-generator.types';
import { Provider } from 'shared-types/api.types';
import { resolveSelectOptions } from '../../ui-component/utils/select-component-handler';
import { withNormalizedPathMeta } from './normalized-component';
import { applyModeOverrides } from './apply-mode-overrides';
import { providerRegistry } from '../../api-providers';
import {
  ToggleableResolution,
  ToggleableScope,
  createToggleableScope,
  resolveToggleable,
} from '../toggleable/toggleable';
import { findUnguardedCelReferences } from '../toggleable/find-unguarded-cel-references';
import { withWidgetTargets } from '../widget-targets';
import { widgetTargetResolvers } from '../../widget-target-resolvers';
import {
  type IgnoredMapField,
  expandKeyValueWidget,
  lintMapFields,
  resolveMapFields,
} from '../map-field';
import { stripEngineKeys } from './strip-engine-keys';

const describeIgnoredMap = ({
  reason,
  overlappingPath,
}: IgnoredMapField): string => {
  switch (reason) {
    case 'overlap':
      return `overlaps the path "${overlappingPath}" of another field, while a map must own its value alone`;
    case 'multi-path':
      return 'has several paths, while an edit reads only the first one';
    default:
      return 'has an itemTemplate this version does not support (only uiType "widget" with widgetType "keyValue")';
  }
};

const describeDegrade = (
  resolution: ToggleableResolution
): string | undefined => {
  switch (resolution.degradeReason) {
    case 'unsafe-key':
      return 'has a section or group key outside [A-Za-z0-9_-], so it cannot name a switch';
    case 'no-fields':
      return 'has no fields with a `path`, so there is nothing to switch off';
    case 'nested':
      return 'is nested inside another toggleable group';
    case 'overlap':
      return `shares the path "${resolution.overlappingPath}" with a field outside it`;
    default:
      return undefined;
  }
};

const preprocessComponent = (
  item: Component | ComponentGroup,
  keyPath: string[],
  scope: ToggleableScope,
  providerObject?: Provider
): Component | ComponentGroup => {
  if (
    (item.uiType === 'group' || item.uiType === 'hidden') &&
    'components' in item
  ) {
    const resolution = resolveToggleable(item, keyPath, scope);
    const warning = describeDegrade(resolution);
    if (warning && import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.warn(
        `[UISchema] Toggleable group "${keyPath.join('.')}" ${warning}; rendering it as bordered.`
      );
    }
    return {
      ...item,
      ...(resolution.degradeReason && { groupType: GroupType.Bordered }),
      _toggleable: resolution.meta,
      components: preprocessComponents(
        item.components,
        keyPath,
        resolution.childScope,
        providerObject
      ),
    };
  }

  const component = item as Component;

  // Dev-time validation: warn if a dataSource references an unregistered provider.
  // TODO: Extract into a dedicated dev-time validation pass
  if (
    import.meta.env.DEV &&
    component.dataSource?.provider &&
    !providerRegistry.has(component.dataSource.provider)
  ) {
    const available = providerRegistry.getAvailableKeys().join(', ');
    // eslint-disable-next-line no-console
    console.warn(
      `[UISchema] Unknown API provider "${component.dataSource.provider}". ` +
        `Available providers: ${available}`
    );
  }
  const normalizedComponent = withNormalizedPathMeta(component);

  // A keyValue item template stands for its row; expand it once here.
  if (isMapComponent(normalizedComponent)) {
    return {
      ...normalizedComponent,
      _row: expandKeyValueWidget(normalizedComponent.itemTemplate),
    };
  }

  if (
    !!providerObject &&
    normalizedComponent.uiType === FieldType.Select &&
    'optionsPath' in normalizedComponent.fieldParams &&
    normalizedComponent.fieldParams.optionsPath
  ) {
    const options = resolveSelectOptions(
      normalizedComponent.fieldParams,
      providerObject
    );

    if (options.length === 0) {
      return normalizedComponent;
    }

    // Build params without the optionsPath fields (replaced by resolved static options)
    const rawFieldParams = normalizedComponent.fieldParams as Extract<
      SelectFieldParams,
      { optionsPath: string }
    >;
    const baseParams = (
      Object.keys(rawFieldParams) as (keyof typeof rawFieldParams)[]
    )
      .filter((k) => k !== 'optionsPath' && k !== 'optionsPathConfig')
      .reduce<Partial<SelectFieldParams>>(
        (acc, k) => ({ ...acc, [k]: rawFieldParams[k] }),
        {}
      ) as SelectFieldParams;

    // Only set defaultValue from resolved options if not already specified in schema
    const defaultValue =
      baseParams.defaultValue !== undefined
        ? baseParams.defaultValue
        : options[0].value;

    return {
      ...normalizedComponent,
      fieldParams: {
        ...baseParams,
        options,
        defaultValue,
      } as SelectFieldParams,
    };
  }

  return normalizedComponent;
};

const preprocessComponents = (
  components: { [key: string]: Component | ComponentGroup },
  parentKeyPath: string[],
  scope: ToggleableScope,
  providerObject?: Provider
): { [key: string]: Component | ComponentGroup } => {
  return Object.fromEntries(
    Object.entries(components).map(([key, item]) => [
      key,
      preprocessComponent(item, [...parentKeyPath, key], scope, providerObject),
    ])
  );
};

export const preprocessSchema = (
  schema: TopologyUISchemas,
  providerObject?: Provider,
  formMode?: FormMode
): TopologyUISchemas => {
  return Object.fromEntries(
    Object.entries(schema).map(([topologyKey, topology]) => {
      if (
        !topology ||
        typeof topology !== 'object' ||
        !('sections' in topology)
      ) {
        return [topologyKey, topology];
      }

      const { sections: declared, stripped } = stripEngineKeys(
        topology.sections
      );
      const withTargets = withWidgetTargets(declared, (widget) =>
        widgetTargetResolvers[widget.widgetType]?.({
          providerObject,
          topology: topologyKey,
        })
      );
      const { sections: withMaps, ignored } = resolveMapFields(withTargets);
      if (import.meta.env.DEV) {
        stripped.forEach(({ keyPath, key }) =>
          // eslint-disable-next-line no-console
          console.warn(
            `[UISchema] "${keyPath}" sets "${key}", a key reserved for the engine; it is ignored.`
          )
        );
        ignored.forEach((field) =>
          // eslint-disable-next-line no-console
          console.warn(
            `[UISchema] Map field "${field.keyPath}" ${describeIgnoredMap(field)}; it is not rendered.`
          )
        );
        lintMapFields(withMaps).forEach(({ keyPath, message }) =>
          // eslint-disable-next-line no-console
          console.warn(`[UISchema] Field "${keyPath}" ${message}.`)
        );
      }
      const scope = createToggleableScope(withMaps);
      const sections = Object.fromEntries(
        Object.entries(withMaps).map(([sectionKey, section]) => [
          sectionKey,
          {
            ...section,
            components: preprocessComponents(
              section.components,
              [sectionKey],
              scope,
              providerObject
            ),
          },
        ])
      );

      if (import.meta.env.DEV) {
        findUnguardedCelReferences(sections).forEach(({ field, group, path }) =>
          // eslint-disable-next-line no-console
          console.warn(
            `[UISchema] A CEL rule on "${field}" reads "${path}" of toggleable group "${group}" without has(${path}); it fails while the group is switched off.`
          )
        );
      }

      return [
        topologyKey,
        {
          ...topology,
          sections: formMode
            ? applyModeOverrides(sections, formMode)
            : sections,
        },
      ];
    })
  ) as TopologyUISchemas;
};
