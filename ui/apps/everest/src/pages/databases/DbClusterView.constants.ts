import { InstanceStatus } from 'shared-types/instance.types';
import { BaseStatus } from 'components/status-field/status-field.types';

export const DB_INSTANCE_STATUS_TO_BASE_STATUS: Record<
  InstanceStatus,
  BaseStatus
> = {
  [InstanceStatus.Creating]: 'creating',
  [InstanceStatus.Running]: 'success',
  [InstanceStatus.Failed]: 'error',
  [InstanceStatus.Deleting]: 'deleting',
  //   [DbClusterStatus.initializing]: 'pending',
  //   [DbClusterStatus.error]: 'error',
  //   [DbClusterStatus.paused]: 'paused',
  //   [DbClusterStatus.pausing]: 'pending',
  //   [DbClusterStatus.ready]: 'success',
  //   [DbClusterStatus.stopping]: 'pending',
  //   [DbClusterStatus.restoring]: 'pending',
  //   [DbClusterStatus.deleting]: 'deleting',
  //   [DbClusterStatus.resizingVolumes]: 'pending',
  //   [DbClusterStatus.creating]: 'creating',
  //   [DbClusterStatus.upgrading]: 'upgrading',
  //   [DbClusterStatus.importing]: 'importing',
};
