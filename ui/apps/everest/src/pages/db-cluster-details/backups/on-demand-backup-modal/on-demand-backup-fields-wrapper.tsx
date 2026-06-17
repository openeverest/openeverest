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

import { MenuItem } from '@mui/material';
import { SelectInput, TextInput } from '@percona/ui-lib';
import { useBackupClassesList } from 'hooks/api/backup-classes/useBackupClasses';
import { useBackupsList } from 'hooks/api/backups/useBackups';
import { useClusterName } from 'hooks/api/useClusterName';
import { useContext, useEffect, useMemo } from 'react';
import { useFormContext } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import { removeUnusedStorages } from '../backups.utils';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import BackupStoragesInput from 'components/backup-storages-input';
import { BackupConfigFields } from 'components/backup-config-fields';
import { BackupFields } from './on-demand-backup-modal.types';
import { ScheduleModalContext } from '../backups.context';

export const OnDemandBackupFieldsWrapper = () => {
  const clusterName = useClusterName();
  const { instanceName = '', namespace = '' } = useParams();
  const { instance } = useContext(ScheduleModalContext);
  const { watch, setValue } = useFormContext();

  const selectedClassName: string = watch(BackupFields.backupClassName);

  const { data: backupClasses = [], isLoading: loadingClasses } =
    useBackupClassesList(clusterName);

  const selectedClass = useMemo(
    () => backupClasses.find((bc) => bc.metadata?.name === selectedClassName),
    [backupClasses, selectedClassName]
  );

  // Filter classes that support this instance's provider.
  const providerType = instance.spec?.provider;
  const instanceClassRef = instance.spec?.backup?.classRef?.name;
  const instanceClass = backupClasses.find(
    (bc) => bc.metadata?.name === instanceClassRef
  );
  const instanceUsesProviderManaged =
    instanceClass?.spec?.executionMode === 'ProviderManaged';

  const availableClasses = backupClasses.filter((bc) => {
    const supported = bc.spec?.supportedProviders;
    if (supported && supported.length > 0 && providerType) {
      if (!supported.includes(providerType)) return false;
    }
    // If the instance already uses a ProviderManaged class,
    // only allow that same PM class or any Job class.
    if (
      instanceUsesProviderManaged &&
      bc.spec?.executionMode === 'ProviderManaged'
    ) {
      return bc.metadata?.name === instanceClassRef;
    }
    return true;
  });

  const maxStorages = selectedClass?.spec?.providerManaged?.limits?.maxStorages;
  const maxSchedulesPerStorage =
    selectedClass?.spec?.providerManaged?.limits?.maxSchedulesPerStorage;

  const instanceStorages = instance.spec?.backup?.storages ?? [];

  const { data: backups = [] } = useBackupsList(
    clusterName,
    namespace,
    instanceName
  );
  const liveInstanceStorages = useMemo(
    () => removeUnusedStorages(instanceStorages, backups),
    [instanceStorages, backups]
  );

  useEffect(() => {
    if (availableClasses.length > 0 && !selectedClassName) {
      setValue(
        BackupFields.backupClassName,
        availableClasses[0].metadata?.name ?? '',
        { shouldValidate: true }
      );
    }
  }, [availableClasses, selectedClassName, setValue]);

  return (
    <>
      <TextInput
        name={BackupFields.name}
        textFieldProps={{
          label: 'Backup name',
        }}
        isRequired
      />
      <SelectInput
        name={BackupFields.backupClassName}
        label="Backup class"
        selectFieldProps={{
          label: 'Backup class',
          disabled: loadingClasses,
        }}
      >
        {availableClasses.map((bc) => (
          <MenuItem key={bc.metadata?.name} value={bc.metadata?.name ?? ''}>
            {bc.spec?.displayName || bc.metadata?.name}
          </MenuItem>
        ))}
      </SelectInput>
      <BackupStoragesInput
        name={BackupFields.storageName}
        namespace={namespace}
        instanceStorages={liveInstanceStorages}
        maxStorages={maxStorages}
        maxSchedulesPerStorage={maxSchedulesPerStorage}
        autoFillProps={{
          isRequired: true,
        }}
      />
      <BackupConfigFields
        backupClass={selectedClass}
        formMode={FormMode.New}
        namespace={namespace}
      />
    </>
  );
};
