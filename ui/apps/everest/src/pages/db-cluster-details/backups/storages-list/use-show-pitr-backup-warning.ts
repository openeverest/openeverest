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

import { useActiveBackupClass } from 'hooks/api/backup-classes/useBackupClasses';
import { useBackupsList } from 'hooks/api/backups/useBackups';
import { useClusterName } from 'hooks/api/useClusterName';
import { Instance } from 'shared-types/api.types';
import { BackupStatus } from 'shared-types/backups.types';
import { hasActiveSchedules } from '../pitr.utils';

// PITR streams transaction logs forward from a full backup, so it only becomes
// usable once a successful backup exists. Enabling is allowed at any time; this
// decides whether to show the "needs a backup" warning — true only when the
// provider supports PITR and the instance has neither an active schedule nor a
// successful backup to stream from. Backups are only queried when there is no
// schedule, since a schedule already clears the warning.
export const useShowPitrBackupWarning = (instance: Instance): boolean => {
  const activeClass = useActiveBackupClass(instance);
  const supportsPITR =
    activeClass?.spec?.providerManaged?.supportsPITR ?? false;

  const storages = instance.spec?.backup?.storages ?? [];
  const scheduled = hasActiveSchedules(storages);

  const clusterName = useClusterName();
  const namespace = instance.metadata?.namespace ?? '';
  const instanceName = instance.metadata?.name ?? '';
  const { data: backups = [] } = useBackupsList(
    clusterName,
    namespace,
    instanceName,
    { enabled: !!instanceName && !scheduled }
  );

  const hasSucceededBackup = backups.some(
    (backup) => backup.status?.state === BackupStatus.SUCCEEDED
  );

  return supportsPITR && !scheduled && !hasSucceededBackup;
};
