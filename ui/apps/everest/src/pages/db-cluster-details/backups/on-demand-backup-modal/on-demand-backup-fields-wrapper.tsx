// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
