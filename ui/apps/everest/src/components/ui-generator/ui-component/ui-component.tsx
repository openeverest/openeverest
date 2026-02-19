import { MenuItem } from '@mui/material';
import {
  Component,
  FieldType,
} from 'components/ui-generator/ui-generator.types';
import React from 'react';
import { useFormContext, get } from 'react-hook-form';
import { muiComponentMap } from '../constants';
import { getMappedParams } from './get-mapped-params';

type ComponentByType<T extends Component['uiType']> = Extract<
  Component,
  { uiType: T }
>;

export type ComponentProps<
  T extends Component['uiType'] = Component['uiType'],
> = {
  item: ComponentByType<T>;
  name: string;
};

function isSelectComponent(
  item: Component
): item is ComponentByType<FieldType.Select> {
  return item.uiType === FieldType.Select;
}

const UIComponent: React.FC<ComponentProps> = ({ item, name }) => {
  const { uiType, fieldParams, validation } = item;
  const methods = useFormContext();
  const errors = methods?.formState?.errors || {};
  //get() is used to access nested error paths like "spec.replica.nodes"
  const errorObj = get(errors, name);
  const error = errorObj?.message as string | undefined;

  const MuiComponent = muiComponentMap[uiType];
  if (!MuiComponent) return null;

  const label = fieldParams?.label || '';

  const mappedProps = getMappedParams(uiType, fieldParams, validation);

  const options = isSelectComponent(item)
    ? item.fieldParams.options.map((option) => (
        <MenuItem key={`${name}-${option.value}`} value={option.value}>
          {option.label}
        </MenuItem>
      ))
    : undefined;

  return (
    <>
      {React.createElement(
        MuiComponent,
        {
          ...mappedProps,
          name,
          label,
          error: !!error,
          formControlProps: { sx: { minWidth: '450px', marginTop: '15px' } },
        },
        options
      )}
    </>
  );
};

export default UIComponent;
