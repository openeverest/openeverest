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
import { useClusterName } from 'hooks/api/useClusterName.ts';
import { useContext, useEffect, useMemo, useRef } from 'react';
import { useFormContext } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import BackupStoragesInput from 'components/backup-storages-input';
import { BackupFields } from './on-demand-backup-modal.types.ts';
import { ScheduleModalContext } from '../backups.context.ts';
import { getSectionExplicitDefaults } from './on-demand-backup-fields-wrapper.utils';

export const OnDemandBackupFieldsWrapper = () => {
  const clusterName = useClusterName();
  const { namespace = '' } = useParams();
  const { instance } = useContext(ScheduleModalContext);
  const { watch, setValue } = useFormContext();
  const appliedDefaultsClassRef = useRef<string>('');

  const selectedClassName: string = watch(BackupFields.backupClassName);

  const { data: backupClasses = [], isLoading: loadingClasses } =
    useBackupClassesList(clusterName);

  const selectedClass = useMemo(
    () => backupClasses.find((bc) => bc.metadata?.name === selectedClassName),
    [backupClasses, selectedClassName]
  );

  const { sections: backupSections } = useBackupClassUiSchema(selectedClass);

  // Filter classes that support this instance's provider.
  const providerType = instance.spec?.provider;
  const availableClasses = backupClasses.filter((bc) => {
    const supported = bc.spec?.supportedProviders;
    if (!supported || supported.length === 0) return true;
    if (!providerType) return true;
    return supported.includes(providerType);
  });

  const maxStorages = selectedClass?.spec?.providerManaged?.limits?.maxStorages;
  const maxSchedulesPerStorage =
    selectedClass?.spec?.providerManaged?.limits?.maxSchedulesPerStorage;

  // Existing schedules from the instance — needed for storage availability filtering.
  const instanceSchedules = useMemo(() => {
    const storages = instance.spec?.backup?.storages ?? [];
    return storages.flatMap((s) =>
      (s.schedules ?? []).map((sched) => ({
        name: sched.name,
        enabled: sched.enabled,
        schedule: sched.cron,
        backupStorageName: s.storageRef?.name ?? '',
        retentionCopies: sched.retentionCopies,
      }))
    );
  }, [instance]);

  useEffect(() => {
    if (availableClasses.length > 0 && !selectedClassName) {
      setValue(
        BackupFields.backupClassName,
        availableClasses[0].metadata?.name ?? ''
      );
    }
  }, [availableClasses, selectedClassName, setValue]);

  useEffect(() => {
    if (
      !selectedClassName ||
      appliedDefaultsClassRef.current === selectedClassName
    ) {
      return;
    }

    const explicitDefaults = getSectionExplicitDefaults(backupSections?.config);

    Object.entries(explicitDefaults).forEach(([fieldName, defaultValue]) => {
      setValue(fieldName, defaultValue, {
        shouldDirty: false,
        shouldTouch: false,
        shouldValidate: false,
      });
    });

    appliedDefaultsClassRef.current = selectedClassName;
  }, [backupSections, selectedClassName, setValue]);

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
      <BackupStoragesInput
        name={BackupFields.storageName}
        namespace={namespace}
        schedules={instanceSchedules}
        maxStorages={maxStorages}
        maxSchedulesPerStorage={maxSchedulesPerStorage}
        autoFillProps={{
          isRequired: true,
          enableFillFirst: true,
        }}
      />
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
