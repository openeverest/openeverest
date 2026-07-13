import { Stack } from '@mui/material';
import { UnknownIcon } from '@percona/ui-lib';
import { BaseStatus, StatusFieldProps } from './status-field.types';
import { STATUS_TO_ICON } from './status-field.utils';

function StatusField<T extends string | number | symbol>({
  status,
  statusMap,
  children,
  dataTestId,
  iconProps,
  stackProps,
  defaultIcon = UnknownIcon,
}: StatusFieldProps<T>) {
  const mappedStatus: BaseStatus = statusMap[status];
  const MappedIcon = STATUS_TO_ICON[mappedStatus] || defaultIcon;

  return (
    <Stack
      direction="row"
      data-testid={`${dataTestId ? `${dataTestId}-` : ''}status`}
      {...stackProps}
      sx={[{
        gap: 1,
        height: 'fit-content',
        alignItems: 'center'
      }, ...(stackProps?.sx
        ? (Array.isArray(stackProps.sx) ? stackProps.sx : [stackProps.sx])
        : [])]}>
      <MappedIcon {...iconProps} />
      {children}
    </Stack>
  );
}

export default StatusField;
