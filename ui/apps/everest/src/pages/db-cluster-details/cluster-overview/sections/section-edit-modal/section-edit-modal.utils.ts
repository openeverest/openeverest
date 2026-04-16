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
import type { KubernetesClusterInfo } from 'shared-types/kubernetes.types';
import type { Instance } from 'shared-types/api.types';
import { getByPath } from 'components/ui-generator/utils/object-path/object-path';

const STORAGE_SIZE_PATH = 'spec.components.engine.storage.size';
const STORAGE_CLASS_PATH = 'spec.components.engine.storage.storageClass';

// TODO should be moved to ui-schema of provider with "tooltip" property or can be a part of hardcoded module of openeverest errors, needs discussions
const DISABLED_DISK_TOOLTIP =
  "You can't change the disk size as the selected storage class doesn't support volume expansion.";

const enrichComponent = (
  item: Component | ComponentGroup,
  overrides: Record<string, { disabled?: boolean; tooltip?: string }>
): Component | ComponentGroup => {
  if (
    (item.uiType === 'group' || item.uiType === 'hidden') &&
    'components' in item
  ) {
    const group = item as ComponentGroup;
    return {
      ...group,
      components: enrichComponents(group.components, overrides),
    };
  }

  const component = item as Component;
  const path =
    'path' in component && typeof component.path === 'string'
      ? component.path
      : undefined;

  if (!path || !overrides[path]) {
    return component;
  }

  const override = overrides[path];
  return {
    ...component,
    fieldParams: {
      ...component.fieldParams,
      ...(override.disabled !== undefined
        ? { disabled: override.disabled }
        : {}),
      ...(override.tooltip !== undefined ? { tooltip: override.tooltip } : {}),
    },
  } as Component;
};

const enrichComponents = (
  components: Record<string, Component | ComponentGroup>,
  overrides: Record<string, { disabled?: boolean; tooltip?: string }>
): Record<string, Component | ComponentGroup> =>
  Object.fromEntries(
    Object.entries(components).map(([key, item]) => [
      key,
      enrichComponent(item, overrides),
    ])
  );

/**
 * Enriches sections with runtime field overrides based on cluster info.
 * Currently checks whether the instance's storage class supports volume expansion
 * and disables the storage size field with a tooltip if it does not.
 */
export const applyRuntimeOverrides = (
  sections: Record<string, Section>,
  instance: Instance,
  clusterInfo?: KubernetesClusterInfo
): Record<string, Section> => {
  const overrides = buildStorageSizeOverrides(instance, clusterInfo);

  if (Object.keys(overrides).length === 0) {
    return sections;
  }

  return Object.fromEntries(
    Object.entries(sections).map(([sectionKey, section]) => [
      sectionKey,
      {
        ...section,
        components: enrichComponents(section.components, overrides),
      },
    ])
  );
};

const buildStorageSizeOverrides = (
  instance: Instance,
  clusterInfo?: KubernetesClusterInfo
): Record<string, { disabled?: boolean; tooltip?: string }> => {
  const instanceData = instance as unknown as Record<string, unknown>;
  const storageClass = getByPath(instanceData, STORAGE_CLASS_PATH);

  if (!storageClass || typeof storageClass !== 'string') {
    return {};
  }

  const storageClassInfo = (clusterInfo?.storageClasses ?? []).find(
    (sc) => sc.metadata.name === storageClass
  );

  if (!storageClassInfo || storageClassInfo.allowVolumeExpansion !== false) {
    return {};
  }

  return {
    [STORAGE_SIZE_PATH]: {
      disabled: true,
      tooltip: DISABLED_DISK_TOOLTIP,
    },
  };
};
