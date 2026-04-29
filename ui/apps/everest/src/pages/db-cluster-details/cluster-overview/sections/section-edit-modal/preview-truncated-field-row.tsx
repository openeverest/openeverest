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

import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';
import { Link, Stack, Typography } from '@mui/material';
import { Messages } from './preview-truncated-field-row.messages';

export type PreviewTruncatedFieldRowProps = {
  label: string;
  value: string;
  dataTestId?: string;
};

const clampedValueSx = {
  display: '-webkit-box',
  WebkitBoxOrient: 'vertical' as const,
  WebkitLineClamp: 2,
  overflow: 'hidden',
  wordBreak: 'break-word' as const,
  whiteSpace: 'pre-wrap' as const,
};

const expandedValueSx = {
  whiteSpace: 'pre-wrap' as const,
  wordBreak: 'break-word' as const,
};

export const PreviewTruncatedFieldRow = ({
  label,
  value,
  dataTestId = 'preview-truncated-field',
}: PreviewTruncatedFieldRowProps) => {
  const [expanded, setExpanded] = useState(false);
  const [hasOverflow, setHasOverflow] = useState(false);
  const valueRef = useRef<HTMLParagraphElement>(null);

  const measureOverflow = useCallback(() => {
    const el = valueRef.current;
    if (!el || expanded) {
      return;
    }
    setHasOverflow(el.scrollHeight > el.clientHeight + 1);
  }, [expanded]);

  useLayoutEffect(() => {
    measureOverflow();
  }, [measureOverflow, value, expanded]);

  useEffect(() => {
    if (expanded) {
      return;
    }
    const el = valueRef.current;
    if (!el || typeof ResizeObserver === 'undefined') {
      return;
    }
    const ro = new ResizeObserver(() => {
      measureOverflow();
    });
    ro.observe(el);
    return () => {
      ro.disconnect();
    };
  }, [expanded, measureOverflow, value]);

  const showValueToggle = !expanded && hasOverflow;
  const valueInteractive = showValueToggle;

  return (
    <Stack
      spacing={0.25}
      data-testid={dataTestId}
      sx={{ alignItems: 'flex-start' }}
    >
      <Typography
        variant="caption"
        color="text.secondary"
        data-testid={`${dataTestId}-label`}
      >
        {label}:
      </Typography>
      <Typography
        ref={valueRef}
        variant="caption"
        color="text.secondary"
        data-testid={`${dataTestId}-value`}
        sx={{
          ...(expanded ? expandedValueSx : clampedValueSx),
          ...(valueInteractive ? { cursor: 'pointer' } : {}),
        }}
        onClick={
          valueInteractive
            ? (e) => {
                e.preventDefault();
                setExpanded(true);
              }
            : undefined
        }
        onKeyDown={
          valueInteractive
            ? (e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  setExpanded(true);
                }
              }
            : undefined
        }
        role={valueInteractive ? 'button' : undefined}
        tabIndex={valueInteractive ? 0 : undefined}
      >
        {value}
      </Typography>
      {showValueToggle && (
        <Link
          component="button"
          type="button"
          onClick={() => setExpanded(true)}
          aria-expanded={false}
          data-testid={`${dataTestId}-toggle`}
          sx={{
            alignSelf: 'flex-end',
            cursor: 'pointer',
            typography: 'caption',
          }}
        >
          {Messages.showMore}
        </Link>
      )}
      {expanded && hasOverflow && (
        <Link
          component="button"
          type="button"
          onClick={() => setExpanded(false)}
          aria-expanded
          data-testid={`${dataTestId}-toggle`}
          sx={{
            alignSelf: 'flex-end',
            cursor: 'pointer',
            typography: 'caption',
          }}
        >
          {Messages.showLess}
        </Link>
      )}
    </Stack>
  );
};
