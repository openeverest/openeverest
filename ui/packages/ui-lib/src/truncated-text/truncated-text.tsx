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
import { Tooltip, Typography } from '@mui/material';
import { type TruncatedTextProps } from './truncated-text.types';

const isOverflowing = (el: HTMLElement | null) =>
  !!el && el.scrollWidth > el.clientWidth;

export const TruncatedText = ({
  text,
  tooltipPlacement = 'right',
  typographyProps,
  dataTestId = 'truncated-text',
}: TruncatedTextProps) => {
  const textRef = useRef<HTMLElement>(null);
  const [tooltipOpen, setTooltipOpen] = useState(false);

  // Measured on open rather than on render so the result reflects the
  // current width, e.g. after a drawer finished expanding.
  const handleTooltipOpen = () =>
    setTooltipOpen(isOverflowing(textRef.current));

  return (
    <Tooltip
      title={text}
      placement={tooltipPlacement}
      arrow
      open={tooltipOpen}
      onOpen={handleTooltipOpen}
      onClose={() => setTooltipOpen(false)}
    >
      <Typography
        ref={textRef}
        noWrap
        data-testid={dataTestId}
        {...typographyProps}
      >
        {text}
      </Typography>
    </Tooltip>
  );
};

export default TruncatedText;
