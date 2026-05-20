import { AutoCompleteAutoFillProps } from 'components/auto-complete-auto-fill/auto-complete-auto-fill.types';
import { BackupStorage } from 'shared-types/backupStorages.types';
import { Schedule } from 'shared-types/dbCluster.types';

export type BackupStoragesInputProps = {
  name?: string;
  namespace: string;
  schedules: Schedule[];
  /**
    * Controls how the backup storage field behaves in different product flows.
    *
    * Typical usage:
    * - On-demand and schedule creation forms: keep default auto-selection of the first
    *   available storage to reduce user actions.
    * - Schedule edit and similar update flows: disable auto-selection to preserve
    *   the storage already saved in the existing configuration.
   */
  autoFillProps?: Partial<AutoCompleteAutoFillProps<BackupStorage>>;
  maxStorages?: number;
  maxSchedulesPerStorage?: number;
  /** Storage names currently active on the instance (instance.spec.backup.storages[].storageRef.name). */
  instanceStorageNames?: string[];
  // TODO: schedules — hideUsedStoragesInSchedules was used for PostgreSQL
  // to hide storages already assigned to other schedules (PG slot limit).
  // Re-enable when schedule feature is implemented.
  // hideUsedStoragesInSchedules?: boolean;
};
