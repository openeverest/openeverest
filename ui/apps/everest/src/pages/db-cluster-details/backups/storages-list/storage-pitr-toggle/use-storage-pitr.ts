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

import { useContext } from 'react';
import {
  useActiveBackupClass,
  useBackupClassUiSchema,
} from 'hooks/api/backup-classes/useBackupClasses';
import { useUpdateDbInstanceWithConflictRetry } from 'hooks/api/db-instances/useUpdateDbInstance';
import { useRBACPermissions } from 'hooks/rbac';
import { ScheduleModalContext } from '../../backups.context';
import { InstanceBackupStorage } from '../../backups.types';
import { PitrParameters } from 'components/pitr-config-modal/pitr-config-modal.types';
import {
  buildPitrPayload,
  countPitrEnabledStorages,
  getPitrBlockReason,
  setStoragePitr,
} from '../../pitr.utils';
import { Messages } from './storage-pitr-toggle.messages';

export const useStoragePitr = (storage: InstanceBackupStorage) => {
  const { instance } = useContext(ScheduleModalContext);
  const activeClass = useActiveBackupClass(instance);
  const providerManaged = activeClass?.spec?.providerManaged;
  const { sections } = useBackupClassUiSchema(activeClass);
  const hasSchema = !!sections?.pitr;

  const { canUpdate } = useRBACPermissions(
    'instances',
    `${instance.metadata?.namespace}/${instance.metadata?.name}`
  );

  const storages = instance.spec?.backup?.storages ?? [];
  const enabled = storage.pitr?.enabled ?? false;
  const supported = providerManaged?.supportsPITR ?? false;

  const maxPitrStorages = providerManaged?.limits?.maxPITREnabledStorages;

  // The only hard block on enabling is the per-provider storage limit; a
  // missing update permission blocks both directions. A missing backup is a
  // non-blocking warning surfaced by StoragesList, not a disabled state here.
  const blockReason = getPitrBlockReason({
    enabledCount: countPitrEnabledStorages(storages),
    maxEnabled: maxPitrStorages,
    storageEnabled: enabled,
  });
  const blockedReason =
    blockReason === 'limit-reached' && maxPitrStorages != null
      ? Messages.limitReached(maxPitrStorages)
      : undefined;
  const reason = canUpdate ? blockedReason : Messages.noPermission;

  const { mutate: updateInstance, isPending } =
    useUpdateDbInstanceWithConflictRetry(instance);

  const patchPitr = (pitr: InstanceBackupStorage['pitr']) => {
    const backup = instance.spec?.backup;
    if (!backup) {
      return;
    }

    updateInstance({
      ...instance,
      spec: {
        ...instance.spec,
        backup: {
          ...backup,
          storages: setStoragePitr(storages, storage.storageRef.name, pitr),
        },
      },
    });
  };

  const setEnabled = (next: boolean) =>
    patchPitr(buildPitrPayload(storage.pitr, { enabled: next }));

  const setParameters = (parameters: PitrParameters | undefined) =>
    patchPitr(buildPitrPayload(storage.pitr, { enabled: true, parameters }));

  return {
    visible: supported,
    enabled,
    disabled: reason !== undefined || isPending,
    reason,
    showConfig: hasSchema,
    configDisabled: !enabled || !canUpdate,
    activeClass,
    currentParameters: storage.pitr?.parameters,
    namespace: instance.metadata?.namespace,
    isPending,
    setEnabled,
    setParameters,
  };
};
