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

import { useEffect, useMemo } from 'react';
import { useFormContext } from 'react-hook-form';
import { DbWizardFormFields } from 'consts';
import { useClusterName } from 'hooks/api/useClusterName';
import {
  useBackupClassesList,
  useBackupClassUiSchema,
} from 'hooks/api/backup-classes/useBackupClasses';
import { BackupClass } from 'shared-types/backups.types';
import { FlattenedSchedule } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog-context.types';
import { getPitrBlockReason } from 'pages/db-cluster-details/backups/pitr.utils';
import {
  BACKUP_CLASS_REF_FIELD,
  BACKUP_PITR_FIELD,
  BACKUP_SCHEDULES_FIELD,
} from '../schedules';
import { WizardPitrConfig, WizardPitrMap } from '../backup-step.types';
import { Messages } from './pitr-block.messages';

export interface WizardPitrStorage {
  name: string;
  enabled: boolean;
  reason?: string;
  showConfig: boolean;
  configDisabled: boolean;
  currentParameters?: Record<string, unknown>;
}

export interface WizardPitrBlock {
  supported: boolean;
  namespace: string;
  backupClass: BackupClass | undefined;
  storages: WizardPitrStorage[];
  hasScheduledStorages: boolean;
  // Provider cap on how many storages may have PITR enabled at once; undefined
  // means no limit.
  maxEnabled?: number;
  setEnabled: (storageName: string, next: boolean) => void;
  setParameters: (
    storageName: string,
    parameters: Record<string, unknown> | undefined
  ) => void;
}

// Drives the wizard's per-storage PITR block off the live form state. Storages
// are the distinct storage names of the currently active backup schedules;
// gating and the config schema come from the selected BackupClass. Writes land
// in the flat `backup.pitr` map (WizardPitrMap), mapped to the spec at submit.
export const usePitrBlock = (): WizardPitrBlock => {
  const { watch, setValue } = useFormContext();
  const clusterName = useClusterName();
  const { data: backupClasses = [] } = useBackupClassesList(clusterName);

  const namespace: string = watch(DbWizardFormFields.k8sNamespace);
  const selectedClassName: string | undefined = watch(BACKUP_CLASS_REF_FIELD);
  const formSchedules: FlattenedSchedule[] | undefined = watch(
    BACKUP_SCHEDULES_FIELD
  );
  const rawPitrMap: WizardPitrMap | undefined = watch(BACKUP_PITR_FIELD);
  const pitrMap: WizardPitrMap = rawPitrMap ?? {};

  const selectedClass = useMemo(
    () => backupClasses.find((bc) => bc.metadata?.name === selectedClassName),
    [backupClasses, selectedClassName]
  );
  const providerManaged = selectedClass?.spec?.providerManaged;
  const supported = providerManaged?.supportsPITR ?? false;
  const maxEnabled = providerManaged?.limits?.maxPITREnabledStorages;
  const { sections } = useBackupClassUiSchema(selectedClass);
  const hasSchema = !!sections?.pitr;

  const scheduledStorageNames = useMemo(
    () =>
      Array.from(
        new Set(
          (formSchedules ?? [])
            .filter((schedule) => schedule.enabled)
            .map((schedule) => schedule.storageName)
        )
      ),
    [formSchedules]
  );

  const enabledCount = scheduledStorageNames.filter(
    (name) => pitrMap[name]?.enabled
  ).length;

  // Strict cleanup: drop PITR entries for storages whose schedule was removed,
  // and clear everything when the selected class doesn't support PITR. A later
  // re-added schedule starts fresh rather than resurrecting a stale toggle.
  useEffect(() => {
    const current = rawPitrMap ?? {};
    const staleKeys = supported
      ? Object.keys(current).filter(
          (name) => !scheduledStorageNames.includes(name)
        )
      : Object.keys(current);
    if (staleKeys.length === 0) {
      return;
    }
    const next: WizardPitrMap = { ...current };
    staleKeys.forEach((name) => delete next[name]);
    setValue(BACKUP_PITR_FIELD, next, {
      shouldDirty: true,
      shouldValidate: false,
    });
  }, [supported, scheduledStorageNames, rawPitrMap, setValue]);

  const writeConfig = (
    storageName: string,
    config: WizardPitrConfig | undefined
  ) => {
    const next: WizardPitrMap = { ...pitrMap };
    if (config) {
      next[storageName] = config;
    } else {
      delete next[storageName];
    }
    setValue(BACKUP_PITR_FIELD, next, {
      shouldDirty: true,
      shouldValidate: false,
    });
  };

  const storages: WizardPitrStorage[] = scheduledStorageNames.map(
    (storageName) => {
      const enabled = pitrMap[storageName]?.enabled ?? false;
      // Rows only exist for storages with an active schedule, so the only
      // block that can apply here is the per-provider enabled-storages limit.
      const blockReason = getPitrBlockReason({
        enabledCount,
        maxEnabled,
        storageEnabled: enabled,
      });
      return {
        name: storageName,
        enabled,
        reason:
          blockReason === 'limit-reached' && maxEnabled != null
            ? Messages.limitReached(maxEnabled)
            : undefined,
        showConfig: hasSchema,
        configDisabled: !enabled,
        currentParameters: pitrMap[storageName]?.parameters,
      };
    }
  );

  return {
    supported,
    namespace,
    backupClass: selectedClass,
    storages,
    hasScheduledStorages: scheduledStorageNames.length > 0,
    maxEnabled,
    setEnabled: (storageName, next) =>
      writeConfig(storageName, { ...pitrMap[storageName], enabled: next }),
    setParameters: (storageName, parameters) =>
      writeConfig(storageName, { enabled: true, parameters }),
  };
};
