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

import type { ReactNode } from 'react';
import React from 'react';
import type {
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';
import UIComponent from 'components/ui-generator/ui-component/ui-component';
import UIGroup from 'components/ui-generator/ui-group/ui-group';
import { generateFieldId } from './generate-field-id';
import { orderComponents } from './order-components';
import {
  DataSourceField,
  hasDataSource,
  type ComponentWithDataSource,
} from 'components/ui-generator/api-providers';
import { ComponentErrorBoundary } from 'components/ui-generator/component-error-boundary';

export type RenderComponentProps = {
  item: Component | ComponentGroup;
  name: string;
};

// Recursively renders UI components and groups.
export const renderComponent = ({
  item,
  name,
}: RenderComponentProps): ReactNode => {
  const fieldName = generateFieldId(item, name);
  const isGroup = item?.uiType === 'group' && 'components' in item;

  const children = isGroup ? (
    orderComponents(
      (item as ComponentGroup).components,
      (item as ComponentGroup).componentsOrder
    ).map(([childKey, childItem]) => {
      const childFieldName = `${fieldName}.${childKey}`;
      return (
        <React.Fragment key={childFieldName}>
          {renderComponent({
            item: childItem,
            name: childFieldName,
          })}
        </React.Fragment>
      );
    })
  ) : hasDataSource(item as Component) ? (
    <ComponentErrorBoundary key={fieldName} componentName={name}>
      <DataSourceField item={item as ComponentWithDataSource} name={fieldName}>
        {(patchedItem: Component) => (
          <UIComponent key={fieldName} item={patchedItem} name={fieldName} />
        )}
      </DataSourceField>
    </ComponentErrorBoundary>
  ) : (
    <UIComponent key={fieldName} item={item as Component} name={fieldName} />
  );

  if (isGroup) {
    return (
      <ComponentErrorBoundary key={fieldName} componentName={name}>
        <UIGroup
          key={fieldName}
          item={item}
          groupType={(item as ComponentGroup).groupType}
          groupParams={(item as ComponentGroup).groupParams}
        >
          {children}
        </UIGroup>
      </ComponentErrorBoundary>
    );
  }

  return (
    <ComponentErrorBoundary key={fieldName} componentName={name}>
      {children}
    </ComponentErrorBoundary>
  );
};
