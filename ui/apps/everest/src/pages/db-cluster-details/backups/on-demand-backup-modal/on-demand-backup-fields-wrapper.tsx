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
import { SelectInput } from '@percona/ui-lib';
import {
  useBackupClassesList,
  useBackupClassUiSchema,
} from 'hooks/api/backup-classes/useBackupClasses.ts';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages.ts';
import { useClusterName } from 'hooks/api/useClusterName.ts';
import { useContext, useEffect } from 'react';
import { useFormContext } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { BackupFields } from './on-demand-backup-modal.types.ts';
import { ScheduleModalContext } from '../backups.context.ts';

export const OnDemandBackupFieldsWrapper = () => {
  const clusterName = useClusterName();
  const { namespace = '' } = useParams();
  const { instance } = useContext(ScheduleModalContext);
  const { watch, setValue, trigger } = useFormContext();

  const selectedClassName: string = watch(BackupFields.backupClassName);
  const selectedStorage: string = watch(BackupFields.storageName);

  const { data: backupClasses = [], isLoading: loadingClasses } =
    useBackupClassesList(clusterName);

  const { data: namespaceStorages = [], isLoading: loadingStorages } =
    useBackupStoragesByNamespace(namespace);

  const { sections: backupSections } = useBackupClassUiSchema(
    clusterName,
    selectedClassName || undefined
  );

  // Filter classes that support this instance's provider.
  const providerType = instance.spec?.provider;
  const availableClasses = backupClasses.filter((bc) => {
    const supported = bc.spec?.supportedProviders;
    if (!supported || supported.length === 0) return true;
    if (!providerType) return true;
    return supported.includes(providerType);
  });

  // Auto-fill storage with first available value if not yet selected.
  useEffect(() => {
    if (!selectedStorage && namespaceStorages.length > 0) {
      setValue(BackupFields.storageName, namespaceStorages[0].name);
      trigger(BackupFields.storageName);
    }
  }, [namespaceStorages, selectedStorage, setValue, trigger]);

  return (
    <>
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
      <SelectInput
        name={BackupFields.storageName}
        label="Storage"
        selectFieldProps={{
          label: 'Storage',
          disabled: loadingStorages,
        }}
      >
        {namespaceStorages.map((storage) => (
          <MenuItem key={storage.name} value={storage.name}>
            {storage.name}
          </MenuItem>
        ))}
      </SelectInput>
      {backupSections && (
        <UIGenerator
          sectionKey="config"
          sections={backupSections}
          formMode={FormMode.New}
          namespace={namespace}
        />
      )}
    </>
  );
};
