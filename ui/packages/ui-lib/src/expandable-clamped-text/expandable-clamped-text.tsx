// everest
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
  useMemo,
  useRef,
  useState,
  type MouseEvent,
} from 'react';
import {
  Button,
  DialogActions,
  DialogContent,
  DialogTitle,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import Dialog from '../dialog';
import { Messages as DefaultMessages } from './expandable-clamped-text.messages';
import {
  type ExpandableClampedTextExpandStrategy,
  type ExpandableClampedTextProps,
} from './expandable-clamped-text.types';

export const DEFAULT_LINE_CLAMP = 2;
export const DEFAULT_SCROLL_MAX_HEIGHT = 120;
const INLINE_CHAR_LIMIT = 255;
const INLINE_LINE_LIMIT = 5;

const linkSx = {
  cursor: 'pointer',
  typography: 'inherit',
  alignSelf: 'flex-start',
  lineHeight: 1.2,
  px: 0,
  py: 0,
  minHeight: 'auto',
};

const ExpandedTextSx = {
  whiteSpace: 'pre-wrap' as const,
  wordBreak: 'break-word' as const,
};

const INLINE_STRATEGY: ExpandableClampedTextExpandStrategy = {
  type: 'inline',
};

export const ExpandableClampedText = ({
  value,
  lineClamp = DEFAULT_LINE_CLAMP,
  dataTestId = 'expandable-clamped-text',
  expandStrategy = INLINE_STRATEGY,
  textTypographyProps,
  linkTypographyProps,
}: ExpandableClampedTextProps) => {
  const [expandedInline, setExpandedInline] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [hasOverflow, setHasOverflow] = useState(false);
  const valueRef = useRef<HTMLDivElement>(null);
  const resizeFrameRef = useRef<number | null>(null);

  const strategyType = expandStrategy.type;
  const isScrollStrategy = strategyType === 'scroll';

  // When inline strategy is used but text is too long, auto-escalate to dialog
  // (opt-out via autoEscalateToDialog: false on the inline strategy)
  const autoEscalate =
    strategyType === 'inline' && (expandStrategy.autoEscalateToDialog ?? true);

  const tooLongForInline =
    autoEscalate &&
    (value.length > INLINE_CHAR_LIMIT ||
      value.split('\n').length > INLINE_LINE_LIMIT);

  const effectiveStrategy = tooLongForInline ? 'dialog' : strategyType;

  const dialogConfig =
    effectiveStrategy === 'dialog'
      ? {
          title:
            (strategyType === 'dialog'
              ? expandStrategy.dialogTitle
              : strategyType === 'inline'
                ? expandStrategy.dialogTitle
                : undefined) ?? DefaultMessages.dialogTitle,
          closeLabel:
            (strategyType === 'dialog'
              ? expandStrategy.closeDialogLabel
              : undefined) ?? DefaultMessages.close,
          props:
            strategyType === 'dialog' ? expandStrategy.dialogProps : undefined,
        }
      : null;

  useEffect(() => {
    setExpandedInline(false);
    setModalOpen(false);
  }, [value, strategyType]);

  const measureOverflow = useCallback(() => {
    const el = valueRef.current;
    if (!el || expandedInline || isScrollStrategy) {
      return;
    }

    setHasOverflow(el.scrollHeight > el.clientHeight + 1);
  }, [expandedInline, isScrollStrategy]);

  useLayoutEffect(() => {
    measureOverflow();
  }, [measureOverflow, value, expandedInline, lineClamp, strategyType]);

  useEffect(() => {
    if (expandedInline || isScrollStrategy) {
      return;
    }

    const el = valueRef.current;
    if (!el || typeof ResizeObserver === 'undefined') {
      return;
    }

    const ro = new ResizeObserver(() => {
      if (resizeFrameRef.current !== null) {
        cancelAnimationFrame(resizeFrameRef.current);
      }
      resizeFrameRef.current = requestAnimationFrame(() => {
        measureOverflow();
      });
    });

    ro.observe(el);

    return () => {
      ro.disconnect();
      if (resizeFrameRef.current !== null) {
        cancelAnimationFrame(resizeFrameRef.current);
        resizeFrameRef.current = null;
      }
    };
  }, [expandedInline, isScrollStrategy, measureOverflow]);

  const clampedSx = useMemo(
    () => ({
      display: '-webkit-box',
      WebkitBoxOrient: 'vertical' as const,
      WebkitLineClamp: lineClamp,
      overflow: 'hidden',
      wordBreak: 'break-word' as const,
      whiteSpace: 'pre-wrap' as const,
    }),
    [lineClamp]
  );

  const shouldShowToggle = hasOverflow && effectiveStrategy !== 'scroll';

  const toggleLabel =
    effectiveStrategy === 'inline' && expandedInline
      ? DefaultMessages.showLess
      : DefaultMessages.showMore;

  const toggleAriaExpanded =
    effectiveStrategy === 'inline'
      ? expandedInline
      : effectiveStrategy === 'dialog'
        ? modalOpen
        : false;

  const handleToggleClick = (e: MouseEvent<HTMLElement>) => {
    e.preventDefault();

    switch (effectiveStrategy) {
      case 'inline': {
        setExpandedInline((prev) => !prev);
        return;
      }
      case 'dialog': {
        setModalOpen(true);
        return;
      }
      case 'navigate': {
        if (expandStrategy.type === 'navigate') {
          expandStrategy.onExpand();
        }
        return;
      }
      case 'scroll':
      default:
        return;
    }
  };

  const contentSx = isScrollStrategy
    ? {
        ...ExpandedTextSx,
        maxHeight: expandStrategy.maxHeight ?? DEFAULT_SCROLL_MAX_HEIGHT,
        overflowY: 'auto' as const,
        width: '100%',
      }
    : {
        ...(expandedInline ? ExpandedTextSx : clampedSx),
      };

  return (
    <>
      <Stack spacing={0.25} sx={{ width: '100%', alignItems: 'flex-start' }}>
        <Typography
          ref={valueRef}
          component="div"
          data-testid={`${dataTestId}-value`}
          variant={textTypographyProps?.variant}
          color={textTypographyProps?.color}
          sx={{
            ...contentSx,
            ...textTypographyProps?.sx,
          }}
        >
          {value}
        </Typography>

        {shouldShowToggle ? (
          <Link
            component="button"
            type="button"
            underline="always"
            variant={
              linkTypographyProps?.variant ??
              textTypographyProps?.variant ??
              'caption'
            }
            color={linkTypographyProps?.color}
            onClick={handleToggleClick}
            aria-expanded={toggleAriaExpanded}
            data-testid={`${dataTestId}-toggle`}
            sx={{ ...linkSx, ...linkTypographyProps?.sx }}
          >
            {toggleLabel}
          </Link>
        ) : null}
      </Stack>

      {dialogConfig && (
        <Dialog
          open={modalOpen}
          onClose={() => setModalOpen(false)}
          fullWidth
          maxWidth="md"
          scroll="paper"
          {...dialogConfig.props}
        >
          <DialogTitle>{dialogConfig.title}</DialogTitle>
          <DialogContent sx={{ pt: 1 }}>
            <Typography
              component="div"
              sx={{
                typography: textTypographyProps?.variant ? undefined : 'body2',
                ...ExpandedTextSx,
              }}
            >
              {value}
            </Typography>
          </DialogContent>
          <DialogActions>
            <Button
              variant="text"
              onClick={() => setModalOpen(false)}
              data-testid={`${dataTestId}-dialog-close`}
            >
              {dialogConfig.closeLabel}
            </Button>
          </DialogActions>
        </Dialog>
      )}
    </>
  );
};

export default ExpandableClampedText;
