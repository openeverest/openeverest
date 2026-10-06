import { SvgIconProps } from '@mui/material';
import {
  ErrorIcon,
  PausedIcon,
  PendingIcon,
  SuccessIcon,
  UnknownIcon,
  WarningIcon,
} from '@percona/ui-lib';
import { BaseStatus } from './status-field.types';

export const STATUS_TO_ICON: Record<
  BaseStatus,
  (props: SvgIconProps) => React.JSX.Element
> = {
  success: SuccessIcon,
  warning: WarningIcon,
  error: ErrorIcon,
  pending: PendingIcon,
  paused: PausedIcon,
  unknown: UnknownIcon,
  deleting: PendingIcon,
  creating: PendingIcon,
  upgrading: PendingIcon,
  importing: PendingIcon,
};
