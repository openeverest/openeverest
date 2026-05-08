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
import type { ExpandableClampedTextProps } from './expandable-clamped-text.types';

export const DEFAULT_INLINE_MAX_LINES = 16;
export const DEFAULT_LINE_CLAMP = 2;

const linkSx = {
  cursor: 'pointer',
  typography: 'inherit',
  alignSelf: 'flex-end',
};

const ExpandedTextSx = {
  whiteSpace: 'pre-wrap' as const,
  wordBreak: 'break-word' as const,
};

const ExpandableClampedText = ({
  value,
  lineClamp = DEFAULT_LINE_CLAMP,
  inlineMaxLines = DEFAULT_INLINE_MAX_LINES,
  dataTestId = 'expandable-clamped-text',
  textTypographyProps,
  linkTypographyProps,
  dialogTitle,
  closeDialogLabel,
  dialogProps,
}: ExpandableClampedTextProps) => {
  const lineCount = useMemo(() => value.split(/\r\n|\r|\n/).length, [value]);
  const useModalForExpand = lineCount > inlineMaxLines;

  const [expandedInline, setExpandedInline] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [hasOverflow, setHasOverflow] = useState(false);
  const valueRef = useRef<HTMLDivElement>(null);
  const measureRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setExpandedInline(false);
    setModalOpen(false);
    setHasOverflow(false);
  }, [value]);

  const measureOverflow = useCallback(() => {
    const el = valueRef.current;
    if (!el || expandedInline) {
      return;
    }

    // Primary path: works for most layouts.
    const fastOverflowCheck = el.scrollHeight > el.clientHeight + 1;
    if (fastOverflowCheck) {
      setHasOverflow(true);
      return;
    }

    // Fallback for cases where -webkit-line-clamp flattens scroll/client heights.
    const measureEl = measureRef.current;
    if (!measureEl) {
      setHasOverflow(false);
      return;
    }
    const computed = window.getComputedStyle(el);
    measureEl.textContent = value;
    measureEl.style.width = `${el.clientWidth}px`;
    measureEl.style.font = computed.font;
    measureEl.style.lineHeight = computed.lineHeight;
    measureEl.style.letterSpacing = computed.letterSpacing;
    measureEl.style.padding = computed.padding;
    measureEl.style.boxSizing = computed.boxSizing;

    const fullHeight = measureEl.getBoundingClientRect().height;

    setHasOverflow(fullHeight > el.clientHeight + 1);
  }, [expandedInline, value]);

  useLayoutEffect(() => {
    measureOverflow();
  }, [measureOverflow, value, expandedInline, lineClamp]);

  useEffect(() => {
    if (expandedInline) {
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
  }, [expandedInline, measureOverflow, value]);

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

  const showExpandInline =
    hasOverflow && !useModalForExpand && !expandedInline;
  const showCollapseInline =
    hasOverflow && !useModalForExpand && expandedInline;
  const showModalExpand = hasOverflow && useModalForExpand;

  const moreLabel = DefaultMessages.showMore;
  const lessLabel = DefaultMessages.showLess;
  const title = dialogTitle ?? DefaultMessages.dialogTitle;
  const closeLbl = closeDialogLabel ?? DefaultMessages.close;

  return (
    <>
      <Stack spacing={0.25} sx={{ width: '100%', alignItems: 'flex-start' }}>
        <Typography
          ref={valueRef}
          component="div"
          data-testid={`${dataTestId}-value`}
          {...textTypographyProps}
          sx={{
            ...(expandedInline ? ExpandedTextSx : clampedSx),
            ...textTypographyProps?.sx,
          }}
        >
          {value}
        </Typography>

        {showExpandInline ? (
          <Link
            component="button"
            type="button"
            underline="always"
            onClick={(e: MouseEvent<HTMLElement>) => {
              e.preventDefault();
              setExpandedInline(true);
            }}
            aria-expanded={false}
            data-testid={`${dataTestId}-toggle`}
            {...linkTypographyProps}
            sx={{ ...linkSx, ...linkTypographyProps?.sx }}
          >
            {moreLabel}
          </Link>
        ) : null}
        {showCollapseInline ? (
          <Link
            component="button"
            type="button"
            underline="always"
            onClick={(e: MouseEvent<HTMLElement>) => {
              e.preventDefault();
              setExpandedInline(false);
            }}
            aria-expanded
            data-testid={`${dataTestId}-toggle`}
            {...linkTypographyProps}
            sx={{ ...linkSx, ...linkTypographyProps?.sx }}
          >
            {lessLabel}
          </Link>
        ) : null}
        {showModalExpand ? (
          <Link
            component="button"
            type="button"
            underline="always"
            onClick={(e: MouseEvent<HTMLElement>) => {
              e.preventDefault();
              setModalOpen(true);
            }}
            aria-expanded={modalOpen}
            data-testid={`${dataTestId}-toggle`}
            {...linkTypographyProps}
            sx={{ ...linkSx, ...linkTypographyProps?.sx }}
          >
            {moreLabel}
          </Link>
        ) : null}
      </Stack>

      <div
        ref={measureRef}
        aria-hidden="true"
        style={{
          position: 'absolute',
          visibility: 'hidden',
          pointerEvents: 'none',
          zIndex: -1,
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
          border: '0',
          margin: '0',
        }}
      />

      <Dialog
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        fullWidth
        maxWidth="md"
        scroll="paper"
        {...dialogProps}
      >
        <DialogTitle>{title}</DialogTitle>
        <DialogContent sx={{ pt: 1, overflow: 'auto', maxHeight: 'calc(90vh - 140px)' }}>
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
            {closeLbl}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

export default ExpandableClampedText;
