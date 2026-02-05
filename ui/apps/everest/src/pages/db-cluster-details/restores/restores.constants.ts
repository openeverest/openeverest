import {
  PG_STATUS,
  PSMDB_STATUS,
  PXC_STATUS,
} from 'shared-types/restores.types';
import { BaseStatus } from 'components/status-field/status-field.types';

export const RESTORE_STATUS_TO_BASE_STATUS: Record<
  PXC_STATUS | PSMDB_STATUS | PG_STATUS,
  BaseStatus
> = {
  Starting: 'pending',
  [PXC_STATUS.STOPPING]: 'pending',
  [PXC_STATUS.RESTORING]: 'pending',
  [PXC_STATUS.STARTING_CLUSTER]: 'pending',
  [PXC_STATUS.PREPARING_CLUSTER]: 'pending',
  [PXC_STATUS.PITR_RECOVERING]: 'pending',
  Failed: 'error',
  Succeeded: 'success',
  [PSMDB_STATUS.WAITING]: 'pending',
  [PSMDB_STATUS.REQUESTED]: 'pending',
  [PSMDB_STATUS.REJECTED]: 'error',
  [PSMDB_STATUS.RUNNING]: 'pending',
  [PSMDB_STATUS.ERROR]: 'error',
  [PSMDB_STATUS.READY]: 'success',
  [PG_STATUS.RUNNING]: 'pending',
};
