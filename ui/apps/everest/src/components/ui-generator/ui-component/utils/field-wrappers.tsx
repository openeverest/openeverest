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

import React from 'react';
import { Box, Tooltip } from '@mui/material';
import {
  Component,
  FieldType,
} from 'components/ui-generator/ui-generator.types';
import { TruncatedValueTooltip } from 'components/ui-generator/ui-component/truncated-value-tooltip';
import { isPasswordField } from 'components/ui-generator/utils/display-value';
import { isNumberLikeField } from './is-number-like-field';

type FieldWrapper = (
  element: React.ReactElement,
  item: Component,
  name: string
) => React.ReactElement;

const shouldCompensateTooltipMargin = (item: Component): boolean => {
  // TextInput-based fields own their top margin; move it onto the tooltip
  // wrapper so the field stays aligned with neighbours.
  return isNumberLikeField(item.uiType) || item.uiType === FieldType.Text;
};

const tooltipWrapper: FieldWrapper = (element, item) => {
  const tooltip = item.fieldParams?.tooltip;
  if (!tooltip) return element;

  const shouldCompensateMargin = shouldCompensateTooltipMargin(item);

  return (
    <Tooltip title={tooltip} placement="top" arrow data-testid="field-tooltip">
      <Box
        sx={[
          {
            display: 'block',
            alignSelf: 'flex-start',
            flex: '1 1 0',
            minWidth: 0,
            width: '100%',
            '& > *': {
              minWidth: 0,
              width: '100%',
            },
          },
          shouldCompensateMargin && {
            mt: 3,
            // TODO: Revisit this when tooltip becomes a first-class ui-schema
            // feature and field spacing is refactored in ui-lib. Number and
            // text inputs currently own their top margin, so the wrapper must
            // temporarily move that spacing to itself to preserve flex layout.
            '& .MuiTextField-root': {
              mt: 0,
            },
            '& .MuiFormControl-root': {
              mt: 0,
            },
          },
        ]}
      >
        {element}
      </Box>
    </Tooltip>
  );
};

const truncatedValueWrapper: FieldWrapper = (element, item, name) => {
  // A tooltip the provider wrote takes the hover, and a masked value stays
  // masked.
  const showsValue =
    item._renderOverrides?.showFullValueOnHover &&
    !item.fieldParams?.tooltip &&
    !isPasswordField(item);
  if (!showsValue) return element;

  return <TruncatedValueTooltip name={name}>{element}</TruncatedValueTooltip>;
};

const fieldWrappers: FieldWrapper[] = [truncatedValueWrapper, tooltipWrapper];

export const applyFieldWrappers = (
  element: React.ReactElement,
  item: Component,
  name: string
): React.ReactElement =>
  fieldWrappers.reduce((el, wrapper) => wrapper(el, item, name), element);
