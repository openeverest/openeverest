import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import { IconButton, Stack, Typography, useTheme } from '@mui/material';
import { useActiveBreakpoint } from 'hooks/utils/useActiveBreakpoint';
import {
  PreviewContentTextProps,
  PreviewSectionProps,
} from './database-preview.types';
import { kebabize } from '@percona/utils';

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
      sx={[{
        pl: 3,
        pt: 1,
        pb: 1,
        pr: 1,
      }, ...(sx ? (Array.isArray(sx) ? sx : [sx]) : []), !hasBeenReached &&
        !active && {
          pt: 0,
          pb: 0,
        }, active &&
        isDesktop && {
          backgroundColor: 'action.hover',
          mb: 1.5,
        }]}
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
