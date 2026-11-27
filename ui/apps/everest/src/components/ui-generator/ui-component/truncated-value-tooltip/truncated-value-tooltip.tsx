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

import { useRef, useState } from 'react';
import { useWatch } from 'react-hook-form';
import { Box } from '@mui/material';
import { OverflowTooltip } from '@percona/ui-lib';
import type { TruncatedValueTooltipProps } from './truncated-value-tooltip.types';

// Shows the whole value of a text input on hover while the input cuts it off.
export const TruncatedValueTooltip = ({
  name,
  children,
}: TruncatedValueTooltipProps) => {
  const value = useWatch({ name });
  const inputRef = useRef<HTMLElement | null>(null);
  // A focused input scrolls with the caret, so the tooltip would only cover
  // the row above while the user types.
  const [editing, setEditing] = useState(false);

  return (
    <OverflowTooltip
      title={!editing && typeof value === 'string' ? value : ''}
      placement="top"
      arrow
      disableFocusListener
      measureRef={inputRef}
    >
      <Box
        ref={(wrapper: HTMLElement | null) => {
          inputRef.current = wrapper?.querySelector('input') ?? null;
        }}
        // The tooltip would name this wrapper after the value, which the
        // input already exposes.
        aria-label={undefined}
        onFocus={() => setEditing(true)}
        onBlur={() => setEditing(false)}
        sx={{
          '& > *': { width: '100%' },
          '& input': { textOverflow: 'ellipsis' },
        }}
      >
        {children}
      </Box>
    </OverflowTooltip>
  );
};
