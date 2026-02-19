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
      return renderComponent({
        item: childItem,
        name: childFieldName,
      });
    })
  ) : (
    <UIComponent item={item as Component} name={fieldName} />
  );

  if (isGroup) {
    return (
      <UIGroup
        key={fieldName}
        item={item}
        groupType={(item as ComponentGroup).groupType}
        groupParams={(item as ComponentGroup).groupParams}
      >
        {children}
      </UIGroup>
    );
  }

  return <React.Fragment key={fieldName}>{children}</React.Fragment>;
};
