// everest
// Copyright (C) 2023 Percona LLC
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

import { DbClusterStatus, Schedule } from 'shared-types/dbCluster.types';
import { DbClusterForNamespaceResult } from '../../hooks/api/db-clusters/useDbClusters';
import { Messages } from './dbClusterView.messages';
import { DbClusterTableElement } from './dbClusterView.types';
import { Backup, BackupStatus } from 'shared-types/backups.types';
import { isProxy } from 'utils/db';
import { DbErrorType } from 'shared-types/dbErrors.types';

const DB_CLUSTER_STATUS_HUMANIFIED: Record<DbClusterStatus, string> = {
  [DbClusterStatus.ready]: Messages.statusProvider.up,
  [DbClusterStatus.error]: Messages.statusProvider.down,
  [DbClusterStatus.initializing]: Messages.statusProvider.initializing,
  [DbClusterStatus.pausing]: Messages.statusProvider.pausing,
  [DbClusterStatus.paused]: Messages.statusProvider.paused,
  [DbClusterStatus.stopping]: Messages.statusProvider.stopping,
  [DbClusterStatus.restoring]: Messages.statusProvider.restoring,
  [DbClusterStatus.deleting]: Messages.statusProvider.deleting,
  [DbClusterStatus.resizingVolumes]: Messages.statusProvider.resizingVolumes,
  [DbClusterStatus.creating]: Messages.statusProvider.creating,
  [DbClusterStatus.upgrading]: Messages.statusProvider.upgrading,
  [DbClusterStatus.importing]: Messages.statusProvider.importing,
};

export const beautifyDbClusterStatus = (
  status: DbClusterStatus,
  conditions?: { type: string }[]
): string => {
  if (
    status === DbClusterStatus.error &&
    conditions?.some((c) => c.type === DbErrorType.ImportFailed)
  ) {
    return Messages.statusProvider.importFailed;
  }
  return (
    DB_CLUSTER_STATUS_HUMANIFIED[status] || Messages.statusProvider.creating
  );
};

export const convertDbClusterPayloadToTableFormat = (
  data: DbClusterForNamespaceResult[]
): DbClusterTableElement[] => {
  const result: DbClusterTableElement[] = [];
  data.forEach((item) => {
    const tableDataForNamespace: DbClusterTableElement[] = item?.queryResult
      ?.isSuccess
      ? item.queryResult?.data.map((cluster) => ({
          namespace: item.namespace,
          status: cluster.status
            ? cluster.status.status
            : DbClusterStatus.creating,
          dbType: cluster.spec.engine.type,
          dbVersion: cluster.spec.engine.version || '',
          backupsEnabled: (cluster.spec.backup?.schedules || []).length > 0,
          databaseName: cluster.metadata.name,
          cpu: cluster.spec.engine.resources?.cpu || '',
          memory: cluster.spec.engine.resources?.memory || '',
          storage: cluster.spec.engine.storage.size,
          nodes: cluster.spec.engine.replicas,
          proxies: isProxy(cluster.spec.proxy)
            ? cluster.spec.proxy.replicas || 0
            : 0,
          proxyCpu: isProxy(cluster.spec.proxy)
            ? cluster.spec.proxy.resources?.cpu || ''
            : '',
          proxyMemory: isProxy(cluster.spec.proxy)
            ? cluster.spec.proxy.resources?.memory || ''
            : '',
          hostName: cluster.status ? cluster.status.hostname : '',
          exposetype: isProxy(cluster.spec.proxy)
            ? cluster.spec.proxy.expose.type
            : undefined,
          port: cluster.status?.port,
          monitoringConfigName:
            cluster.spec.monitoring?.monitoringConfigName ?? '',
          raw: cluster,
        }))
      : [];
    result.push(...tableDataForNamespace);
  });
  return result;
};

export const getLastBackupTimeDiff = (lastBackup: Date): string => {
  const diffInSeconds = Math.round(
    (new Date().getTime() - lastBackup.getTime()) / 1000
  );

  const days = Math.floor(diffInSeconds / 3600 / 24);
  const hours = Math.floor((diffInSeconds - days * 3600 * 24) / 3600) % 24;
  const minutes = Math.floor((diffInSeconds - hours * 3600) / 60) % 60;
  const seconds = (diffInSeconds - minutes * 60) % 60;

  if (days > 0) {
    return `${days}${Messages.lastBackup.days} ${
      hours > 0 ? hours + Messages.lastBackup.hours : ''
    } 
      ${Messages.lastBackup.ago}`;
  }

  if (hours > 0) {
    return `${hours}${Messages.lastBackup.hours} ${
      minutes > 0 ? minutes + Messages.lastBackup.minutes : ''
    } 
     ${Messages.lastBackup.ago}`;
  }

  if (minutes > 0) {
    return `${minutes} ${Messages.lastBackup.minutes} ${Messages.lastBackup.ago}
    `;
  }

  return `${seconds} ${Messages.lastBackup.seconds} ${Messages.lastBackup.ago}
    `;
};

export const getLastBackupStatus = (
  backups: Backup[],
  schedules: Schedule[]
) => {
  if (!backups.length) {
    if (!schedules.length) {
      return Messages.lastBackup.inactive;
    }
    return Messages.lastBackup.scheduled;
  }

  const filteredBackups = backups.filter(
    (backup) => backup.state !== BackupStatus.FAILED
  );

  const lastBackup = filteredBackups[filteredBackups.length - 1];

  if (!lastBackup) {
    if (!schedules.length) {
      return Messages.lastBackup.inactive;
    }
    return Messages.lastBackup.scheduled;
  }

  if (lastBackup.state === BackupStatus.IN_PROGRESS) {
    return Messages.lastBackup.pending;
  }

  if (lastBackup.state === BackupStatus.UNKNOWN) {
    return Messages.lastBackup.notStarted;
  }
};

export const sortBackupsByTime = (backups: Backup[]) => {
  return backups.sort((b1, b2) => {
    const date1 = b1?.completed ? new Date(b1.completed) : new Date();
    const date2 = b2?.completed ? new Date(b2.completed) : new Date();
    return date1.getTime() - date2.getTime();
  });
};
