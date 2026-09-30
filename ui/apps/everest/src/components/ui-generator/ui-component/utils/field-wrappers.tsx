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
import { Box, IconButton, Tooltip } from '@mui/material';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import {
  Component,
  FieldType,
} from 'components/ui-generator/ui-generator.types';

type FieldWrapper = (
  element: React.ReactElement,
  item: Component
) => React.ReactElement;

const shouldCompensateTooltipMargin = (item: Component): boolean => {
  return item.uiType === FieldType.Number || item.uiType === FieldType.Text;
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

const infoWrapper: FieldWrapper = (element, item) => {
  const info = item.fieldParams?.info;
  if (!info) return element;

  const shouldCompensateMargin =
    item.uiType === FieldType.Number || item.uiType === FieldType.Text;

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        flex: '1 1 0',
        minWidth: 0,
        width: '100%',
      }}
    >
      <Box
        sx={{
          flex: 1,
          minWidth: 0,
          ...(shouldCompensateMargin && {
            '& > *': { minWidth: 0, width: '100%' },
          }),
        }}
      >
        {element}
      </Box>
      <Tooltip title={info} placement="right" arrow>
        <IconButton
          size="small"
          data-testid="field-info-button"
          aria-label="Field information"
          sx={{
            flexShrink: 0,
            color: 'action.active',
            p: 0.5,
            // Align the icon visually with the input control area.
            // Text/number fields own a marginTop of ~15px via formControlProps;
            // add the same offset so the icon sits beside the label row.
            ...(shouldCompensateMargin && { mt: '15px' }),
          }}
        >
          <InfoOutlinedIcon fontSize="small" />
        </IconButton>
      </Tooltip>
    </Box>
  );
};

const fieldWrappers: FieldWrapper[] = [tooltipWrapper, infoWrapper];

export const applyFieldWrappers = (
  element: React.ReactElement,
  item: Component
): React.ReactElement =>
  fieldWrappers.reduce((el, wrapper) => wrapper(el, item), element);
