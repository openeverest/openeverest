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

import { Stack, Typography } from '@mui/material';
import { ExpandableClampedText } from '@percona/ui-lib';
import { PreviewContentText } from '../preview-section';
import { orderComponents } from 'components/ui-generator/utils/component-renderer';
import {
  Component,
  ComponentGroup,
  FieldType,
} from 'components/ui-generator/ui-generator.types';
import { getValueByPath } from 'components/ui-generator/ui-component/utils/get-value-by-path';

// coerceNumberInputValue leaves an unparsable string (e.g. "1a" typed mid-edit)
// in place instead of a number, so a Number field's value isn't always numeric.
const isFiniteNumericValue = (value: unknown): boolean => {
  if (typeof value === 'number') {
    return Number.isFinite(value);
  }

  if (typeof value === 'string') {
    const trimmed = value.trim();
    return trimmed !== '' && Number.isFinite(Number(trimmed));
  }

  return false;
};

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
    const badge = leafComponent.fieldParams?.badge;
    // Matches the badge conditions the input applies in build-field-props.tsx:
    // an endAdornment for Number and Select fields.
    const isBadgeableType =
      leafComponent.uiType === FieldType.Number ||
      leafComponent.uiType === FieldType.Select;
    const hasDisplayableValue =
      leafComponent.uiType === FieldType.Number
        ? isFiniteNumericValue(value)
        : value !== '';
    const showBadge = !!badge && isBadgeableType && hasDisplayableValue;
    displayValue = showBadge ? `${value} ${badge}` : String(value);
  }

  const uniqueKey = `${parentPrefix || ''}:${primaryPath || componentKey}`;

  const isMultilineText =
    leafComponent.uiType === FieldType.Text &&
    !!leafComponent.fieldParams?.multiline;

  if (isMultilineText && displayValue !== '-') {
    return (
      <Stack
        key={uniqueKey}
        spacing={0.25}
        data-testid="preview-truncated-field"
        sx={{ alignItems: 'flex-start', width: '100%' }}
      >
        <Typography
          variant="caption"
          data-testid="preview-truncated-field-label"
          sx={{
            color: 'text.secondary',
          }}
        >
          {label}:
        </Typography>
        <ExpandableClampedText
          value={displayValue}
          dataTestId="preview-truncated-field"
          expandStrategy={{
            type: 'inline',
            dialogTitle: label,
            autoEscalateToDialog: true,
          }}
          textTypographyProps={{
            variant: 'caption',
            color: 'text.secondary',
          }}
        />
      </Stack>
    );
  }

  return (
    <PreviewContentText key={uniqueKey} text={`${label}: ${displayValue}`} />
  );
};
