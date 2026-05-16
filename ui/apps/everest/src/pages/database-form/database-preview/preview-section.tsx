import { useLayoutEffect, useRef, useState } from 'react';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import { IconButton, Link, Stack, Typography, useTheme } from '@mui/material';
import { useActiveBreakpoint } from 'hooks/utils/useActiveBreakpoint';
import {
  PreviewContentTextProps,
  PreviewSectionProps,
  TruncatedPreviewTextProps,
} from './database-preview.types';
import { kebabize } from '@percona/utils';
import { Messages } from './database.preview.messages';

export const PreviewSection = ({
  title,
  order,
  onEditClick,
  children,
  hasBeenReached = false,
  active = false,
  disabled = false,
  hasError = false,
  sx,
  ...stackProps
}: PreviewSectionProps) => {
  const theme = useTheme();
  const showEdit = !active && hasBeenReached;
  const { isDesktop } = useActiveBreakpoint();
  const kebabizedTitle = kebabize(title.replace(/\s/g, ''));

  return (
    <Stack
      data-testid={`section-${kebabize(title).replaceAll(' ', '')}`}
      sx={{
        pl: 3,
        pt: 1,
        pb: 1,
        pr: 1,
        ...(!hasBeenReached &&
          !active && {
            pt: 0,
            pb: 0,
          }),
        ...(active &&
          isDesktop && {
            backgroundColor: 'action.hover',
            mb: 1.5,
          }),
        ...sx,
      }}
      {...stackProps}
    >
      <Stack direction="row">
        <Typography
          variant={hasBeenReached ? 'sectionHeading' : 'caption'}
          color={hasBeenReached ? 'text.primary' : 'text.disabled'}
          sx={{
            position: 'relative',
            ml: -2,
          }}
        >
          {`${order}. ${title}`}
          {showEdit && (
            <IconButton
              // Absolute position to avoid the button's padding from interfering with the spacing
              sx={{
                position: 'absolute',
                top: theme.spacing(-1),
              }}
              color="primary"
              size="small"
              disabled={disabled}
              onClick={onEditClick}
              data-testid={`button-edit-preview-${kebabizedTitle}`}
            >
              <EditOutlinedIcon
                sx={{
                  verticalAlign: 'text-bottom',
                }}
                data-testid={`edit-section-${order}`}
              />
            </IconButton>
          )}
        </Typography>
        {hasError && (
          <ErrorOutlineIcon
            data-testid={`preview-error-${kebabizedTitle}`}
            color="error"
            fontSize="small"
            sx={{
              position: 'relative',
              bottom: '2px',
              ml: 'auto',
            }}
          />
        )}
      </Stack>
      {hasBeenReached && children}
    </Stack>
  );
};

export const PreviewContentText = ({
  text,
  dataTestId,
  ...typographyProps
}: PreviewContentTextProps) => (
  <Typography
    variant="caption"
    color="text.secondary"
    data-testid={
      dataTestId ? `${dataTestId}-preview-content` : 'preview-content'
    }
    {...typographyProps}
  >
    {text}
  </Typography>
);

export const TruncatedPreviewText = ({
  text,
  dataTestId,
  maxLines = 2,
  ...typographyProps
}: TruncatedPreviewTextProps) => {
  const [expanded, setExpanded] = useState(false);
  const [isOverflowing, setIsOverflowing] = useState(false);
  const textRef = useRef<HTMLSpanElement>(null);
  const testId = dataTestId
    ? `${dataTestId}-preview-content`
    : 'preview-content';

  // Reset expanded state when text prop changes
  useLayoutEffect(() => {
    setExpanded(false);
  }, [text]);

  useLayoutEffect(() => {
    const el = textRef.current;
    if (!el) return;

    const checkOverflow = () => {
      // Compare scrollHeight to clientHeight to determine if text overflows
      setIsOverflowing(el.scrollHeight > el.clientHeight);
    };

    // Initial check
    checkOverflow();

    // Re-check on dimension changes
    let observer: ResizeObserver | undefined;
    if (typeof ResizeObserver !== 'undefined') {
      observer = new ResizeObserver(checkOverflow);
      observer.observe(el);
    }

    return () => {
      if (observer) {
        observer.disconnect();
      }
    };
  }, [text, maxLines]);

  return (
    <Stack
      spacing={0}
      data-testid={
        dataTestId
          ? `${dataTestId}-truncated-wrapper`
          : 'truncated-wrapper'
      }
    >
      <Typography
        ref={textRef}
        variant="caption"
        color="text.secondary"
        data-testid={testId}
        sx={{
          ...(!expanded && {
            display: '-webkit-box',
            WebkitLineClamp: maxLines,
            WebkitBoxOrient: 'vertical',
            overflow: 'hidden',
          }),
          wordBreak: 'break-word',
          whiteSpace: 'pre-wrap',
        }}
        {...typographyProps}
      >
        {text}
      </Typography>
      {isOverflowing && (
        <Link
          component="button"
          variant="caption"
          onClick={() => setExpanded((prev) => !prev)}
          aria-expanded={expanded}
          data-testid={
            dataTestId
              ? `${dataTestId}-truncated-toggle`
              : 'truncated-toggle'
          }
          sx={{
            alignSelf: 'flex-start',
            cursor: 'pointer',
            mt: 0.25,
          }}
        >
          {expanded ? Messages.showLess : Messages.showMore}
        </Link>
      )}
    </Stack>
  );
};
