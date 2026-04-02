import { DbType } from '@percona/types';
import { AutoCompleteAutoFill } from 'components/auto-complete-auto-fill/auto-complete-auto-fill';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages';
import { useFormContext } from 'react-hook-form';
import { useEffect } from 'react';
import { DbCluster } from 'shared-types/dbCluster.types';
import { dbEngineToDbType } from '@percona/utils';
import { PitrEditFields } from './edit-pitr.types';
import { Messages } from './edit-pitr.messages';

const PitrStorage = ({ dbCluster }: { dbCluster: DbCluster }) => {
  const { watch, setValue } = useFormContext();
  const [pitrEnabled, pitrStorageLocation] = watch([
    PitrEditFields.enabled,
    PitrEditFields.storageLocation,
  ]);

  const { data: backupStorages = [], isFetching: loadingBackupStorages } =
    useBackupStoragesByNamespace(dbCluster.metadata.namespace);

  const dbType = dbEngineToDbType(dbCluster.spec.engine.type);

  useEffect(() => {
    if (
      pitrStorageLocation !== null &&
      !pitrStorageLocation.name &&
      backupStorages[0]
    ) {
      setValue(PitrEditFields.storageLocation, backupStorages[0]);
    }
  }, [pitrStorageLocation, backupStorages]);

  if (!pitrEnabled) {
    return null;
  }

  if (dbType === DbType.Mysql) {
    return (
      <AutoCompleteAutoFill
        name={PitrEditFields.storageLocation}
        label={Messages.enablePITR}
        loading={loadingBackupStorages}
        options={backupStorages}
        isRequired
        enableFillFirst
      />
    );
  }
};
export default PitrStorage;
