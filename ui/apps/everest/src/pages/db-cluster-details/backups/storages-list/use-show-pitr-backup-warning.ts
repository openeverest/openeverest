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

// PITR streams transaction logs forward from a full backup, so it only becomes
// usable once a successful backup completes. Enabling is allowed at any time;
// this decides whether to show the "needs a backup" warning — true whenever the
// provider supports PITR and the instance has no successful backup yet. A
// schedule does not clear it, since a schedule that hasn't produced a completed
// backup still leaves PITR unusable.
export const useShowPitrBackupWarning = (instance: Instance): boolean => {
  const activeClass = useActiveBackupClass(instance);
  const supportsPITR =
    activeClass?.spec?.providerManaged?.supportsPITR ?? false;

  const clusterName = useClusterName();
  const namespace = instance.metadata?.namespace ?? '';
  const instanceName = instance.metadata?.name ?? '';
  const { data: backups = [] } = useBackupsList(
    clusterName,
    namespace,
    instanceName,
    { enabled: !!instanceName }
  );

  const hasSucceededBackup = backups.some(
    (backup) => backup.status?.state === BackupStatus.SUCCEEDED
  );

  return supportsPITR && !hasSucceededBackup;
};
