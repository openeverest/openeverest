import { capitalize, Typography, TypographyProps } from '@mui/material';
import StatusField from 'components/status-field';
import { COMPONENT_STATUS, CONTAINER_STATUS } from '../components.constants';
import {
  BaseStatus,
  StatusFieldProps,
} from 'components/status-field/status-field.types';

const ComponentStatus = <T extends COMPONENT_STATUS | CONTAINER_STATUS>({
  status,
  statusMap,
  typographyProps,
}: {
  status: T;
  statusMap: Record<T, BaseStatus>;
  typographyProps?: TypographyProps;
} & StatusFieldProps<T>) => (
  <StatusField status={status} statusMap={statusMap}>
    <Typography
      variant="body2"
      {...typographyProps}
      sx={[{
        fontWeight: "bold"
      }, ...(typographyProps?.sx
        ? (Array.isArray(typographyProps.sx) ? typographyProps.sx : [typographyProps.sx])
        : [])]}>
      {capitalize(status)}
    </Typography>
  </StatusField>
);

export default ComponentStatus;
