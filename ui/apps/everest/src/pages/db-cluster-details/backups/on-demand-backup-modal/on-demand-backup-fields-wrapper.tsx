import LogicalPhysicalRadioGroup from 'components/logical-physical-radio-group';
import { useContext, useEffect } from 'react';
import { useFormContext } from 'react-hook-form';
// import { Messages } from '../../db-cluster-details.messages.ts';
import { BackupFields } from './on-demand-backup-modal.types.ts';
import { DbEngineType } from '@percona/types';
import { ScheduleModalContext } from '../backups.context.ts';
import BackupStoragesInput from 'components/backup-storages-input';
import { dbEngineToDbType } from '@percona/utils';

export const OnDemandBackupFieldsWrapper = () => {
  const { dbCluster } = useContext(ScheduleModalContext);
  const {
    metadata: { namespace },
    status,
    spec: {
      engine: { type },
      backup,
    },
  } = dbCluster;
  const { setValue, trigger } = useFormContext();
  const dbClusterActiveStorage = status?.activeStorage;

  useEffect(() => {
    if (dbClusterActiveStorage) {
      setValue(BackupFields.storageLocation, {
        name: dbClusterActiveStorage,
      });
      trigger(BackupFields.storageLocation);
    }
  }, [dbClusterActiveStorage]);

  return (
    <>
      {type === DbEngineType.PSMDB && <LogicalPhysicalRadioGroup />}
      {/* <Typography variant="sectionHeading" mt={3} mb={1}>
        {Messages.onDemandBackupModal.backupDetails}
      </Typography> */}
      {/* <TextInput
        name={BackupFields.name}
        textFieldProps={{
          label: Messages.onDemandBackupModal.backupName,
        }}
        isRequired
      /> */}
      <BackupStoragesInput
        dbClusterName={dbCluster.metadata.name}
        namespace={namespace}
        dbType={dbEngineToDbType(type)}
        schedules={backup?.schedules || []}
        autoFillProps={{
          enableFillFirst: true,
          isRequired: true,
          disabled: !!dbClusterActiveStorage,
        }}
      />
    </>
  );
};
