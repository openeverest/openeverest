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

import { useEffect, useMemo, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import {
  useBackupImports,
  useCreateBackupImport,
} from 'hooks/api/backup-imports';
import { getNamespaceBackupsKeyPrefix } from 'hooks/api/backups/useBackups';
import { useClusterName } from 'hooks/api/useClusterName';
import { ScanProgress, ScanTarget } from '../database-import.types';
import {
  findLatestBackupImport,
  isBackupImportFinished,
} from '../database-import.utils';

const POLL_INTERVAL_MS = 2000;
// A scan still unstarted by then likely has no controller to run it.
const STALLED_AFTER_MS = 2 * 60 * 1000;

interface BackupScan extends ScanProgress {
  scan: () => void;
  canScan: boolean;
  hasScanned: boolean;
}

// The scan state comes from the cluster, so it survives leaving the page and
// is shared with other tabs and users.
export const useBackupScan = ({
  namespace,
  storageName,
  backupClassName,
}: ScanTarget & { namespace: string }): BackupScan => {
  const clusterName = useClusterName();
  const queryClient = useQueryClient();
  const target = useMemo(
    () => ({ storageName, backupClassName }),
    [storageName, backupClassName]
  );

  const { data: imports = [] } = useBackupImports(clusterName, namespace, {
    refetchInterval: (query) => {
      const latest = findLatestBackupImport(
        query.state.data?.items ?? [],
        target
      );
      // A failed poll stops here; the next scan restarts it.
      return query.state.status !== 'error' &&
        !!latest &&
        !isBackupImportFinished(latest)
        ? POLL_INTERVAL_MS
        : false;
    },
  });
  const { mutate: createBackupImport, isPending: isStarting } =
    useCreateBackupImport(clusterName, namespace);

  const latest = useMemo(
    () => findLatestBackupImport(imports, target),
    [imports, target]
  );
  const latestName = latest?.metadata?.name;
  const state = latest?.status?.state;
  const startedAt = latest?.metadata?.creationTimestamp;
  const isScanning = !!latest && !isBackupImportFinished(latest);

  const [stalledImportName, setStalledImportName] = useState<string>();
  useEffect(() => {
    if (!isScanning || !latestName || !startedAt) {
      return undefined;
    }
    const timer = setTimeout(
      () => setStalledImportName(latestName),
      Math.max(0, Date.parse(startedAt) + STALLED_AFTER_MS - Date.now())
    );
    return () => clearTimeout(timer);
  }, [isScanning, latestName, startedAt]);
  const isStalled = isScanning && stalledImportName === latestName;

  // Refresh the backups only when a scan finishes while the page follows it.
  const followedScan = useRef<{ name?: string; state?: string }>({});
  useEffect(() => {
    const previous = followedScan.current;
    followedScan.current = { name: latestName, state };
    if (
      state === 'Succeeded' &&
      previous.name === latestName &&
      previous.state !== 'Succeeded'
    ) {
      queryClient.invalidateQueries({
        queryKey: getNamespaceBackupsKeyPrefix(clusterName, namespace),
      });
    }
  }, [latestName, state, clusterName, namespace, queryClient]);

  const scan = () =>
    createBackupImport({
      classRef: { name: backupClassName },
      storageRef: { name: storageName },
    });

  return {
    scan,
    // An erroring or stalled scan may never finish, so it doesn't block a new one.
    canScan: !isStarting && (!isScanning || isStalled || state === 'Error'),
    hasScanned: !!latest,
    isScanning: isStarting || isScanning,
    isStalled,
    state,
    message: latest?.status?.message,
    discoveredCount: latest?.status?.discoveredCount,
    startedAt,
  };
};
