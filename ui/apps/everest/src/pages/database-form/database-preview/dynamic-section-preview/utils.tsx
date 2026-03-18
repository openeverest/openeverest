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

import { PreviewContentText } from '../preview-section';
import { orderComponents } from 'components/ui-generator/utils/component-renderer';
import {
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';
import { getValueByPath } from 'components/ui-generator/ui-component/utils/get-value-by-path';

const getPrimaryPath = (
  path: Component['path'] | undefined
): string | undefined => {
  if (!path) {
    return undefined;
  }

  if (typeof path === 'string') {
    return path;
  }

  // For multipath fields preview reads from the first canonical path.
  return path.find((p): p is string => typeof p === 'string' && !!p);
};

//TODO describe types
export const renderComponent = (
  componentKey: string,
  component: Component | ComponentGroup,
  formValues: Record<string, unknown>,
  parentPrefix = ''
): React.ReactNode => {
  if (!component) return null;

  if (component.uiType === 'group' && 'components' in component) {
    return orderComponents(component.components, component.componentsOrder).map(
      ([subKey, subComp]) =>
        renderComponent(
          `${componentKey}.${subKey}`,
          subComp,
          formValues,
          parentPrefix ? `${parentPrefix}.${componentKey}` : componentKey
        )
    );
  }

  const leafComponent = component as Component;
  const primaryPath = getPrimaryPath(leafComponent.path);
  const value = primaryPath
    ? getValueByPath(formValues, primaryPath)
    : undefined;
  const label = leafComponent.fieldParams?.label || componentKey;

  let displayValue: string = '-';

  if (value === null || value === undefined) {
    displayValue = '-';
  } else if (typeof value === 'boolean') {
    displayValue = value ? 'Enabled' : 'Disabled';
  } else if (typeof value === 'object' && !Array.isArray(value)) {
    displayValue = JSON.stringify(value);
  } else {
    displayValue = String(value);
  }

  const uniqueKey = `${parentPrefix || ''}:${primaryPath || componentKey}`;
  return (
    <PreviewContentText key={uniqueKey} text={`${label}: ${displayValue}`} />
  );
};
