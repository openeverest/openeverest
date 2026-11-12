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

import { useCallback, useMemo, useRef, useState } from 'react';
import { MRT_ColumnDef } from 'material-react-table';
import { Button } from '@mui/material';
import { Table } from '@percona/ui-lib';
import semverCoerce from 'semver/functions/coerce';
import {
  DbEngineToolStatus,
  OperatorUpgradePendingAction,
} from 'shared-types/dbEngines.types';
import { DbCluster, Spec } from 'shared-types/dbCluster.types';
import { ClusterStatusTableProps } from './types';
import { useDbClusters } from 'hooks/api/db-clusters/useDbClusters';
import { useDbInstanceList } from 'hooks/api/db-instances';
import { useNow } from 'hooks/utils/useNow';
import {
  countNeedingAttention,
  getInstanceStatusColumns,
  INSTANCE_STATUS_SORTING,
  InstanceStatusRow,
  NeedsAttentionFilter,
} from 'components/instance-status';
import { getInstanceStatusDetails } from 'utils/instance-conditions';
import { getInstanceStatusDisplay } from 'utils/instance-status';
import UpdateCrDialog from './update-cr-dialog';
import UpdateEngineDialog from './update-engine-dialog';

type EnhancedDbList = OperatorUpgradePendingAction & {
  db?: DbCluster;
  // How the database is doing; absent until its Instance is loaded.
  instance?: InstanceStatusRow;
};

const ClusterStatusTable = ({
  namespace,
  pendingActions,
  dbEngines,
}: ClusterStatusTableProps) => {
  const dbNames = pendingActions.map((db) => db.name);
  const { data: dbClusters = [] } = useDbClusters(namespace, {
    select: (clusters) =>
      clusters.items.filter((cluster) =>
        dbNames.includes(cluster.metadata.name)
      ),
    enabled: !!namespace && !!pendingActions.length,
  });
  const { data: instances = [] } = useDbInstanceList(namespace, {
    enabled: !!namespace && !!pendingActions.length,
  });
  const now = useNow();
  const [openUpdateCrDialog, setOpenUpdateCrDialog] = useState(false);
  const [openUpdateEngineDialog, setOpenUpdateEngineDialog] = useState(false);
  const selectedDbCluster = useRef<DbCluster>();
  const selectedDbEngineVersion = useRef<string>();
  const enhancedDbList: EnhancedDbList[] = useMemo(
    () =>
      pendingActions.map((action) => {
        const db = dbClusters.find(
          (cluster) => cluster.metadata.name === action.name
        );
        const instance = instances.find(
          (item) => item.metadata?.name === action.name
        );

        return {
          ...action,
          db,
          instance: instance && {
            name: action.name,
            status: getInstanceStatusDisplay(instance, now),
            notices: getInstanceStatusDetails(instance).markers,
          },
        };
      }),
    [dbClusters, instances, now, pendingActions]
  );

  const onDbClick = useCallback(
    (db: OperatorUpgradePendingAction) => {
      const { pendingTask } = db;
      selectedDbCluster.current = dbClusters.find(
        (cluster) => cluster.metadata.name === db.name
      );

      if (!selectedDbCluster.current?.metadata.name) {
        return;
      }

      if (pendingTask === 'restart') {
        if (selectedDbCluster.current.status?.recommendedCRVersion) {
          setOpenUpdateCrDialog(true);
        }
      } else if (pendingTask === 'upgradeEngine') {
        // We try to find the version in the message
        const msg = db.message;
        const coercedVersion = semverCoerce(msg, {
          includePrerelease: true,
        });

        if (coercedVersion) {
          // We keep the version exactly as it is in the message
          msg.split(' ').forEach((original) => {
            if (semverCoerce(original)) {
              selectedDbEngineVersion.current = original;
            }
          });
          setOpenUpdateEngineDialog(true);
        } else {
          // Couldn't find the version in the message. Try to update to the latest version with same major as current and recommended
          const currenEngineVersion = semverCoerce(
            selectedDbCluster.current.spec.engine.version,
            { includePrerelease: true }
          );

          if (currenEngineVersion) {
            const currentMajor = currenEngineVersion.major;
            const dbEngine = dbEngines.find(
              (engine) =>
                engine.type === selectedDbCluster.current?.spec.engine.type
            );
            const availableVersions = (dbEngine?.availableVersions.engine || [])
              .filter(({ version, status }) => {
                const semver = semverCoerce(version);
                return (
                  status === DbEngineToolStatus.RECOMMENDED &&
                  semver?.major === currentMajor
                );
              })
              .map(({ version }) => version);
            const sortedVersions = availableVersions.sort();
            if (sortedVersions.length) {
              selectedDbEngineVersion.current =
                sortedVersions[sortedVersions.length - 1];
              setOpenUpdateEngineDialog(true);
            }
          }
        }
      }
    },
    [dbClusters, dbEngines]
  );

  const columns = useMemo<MRT_ColumnDef<EnhancedDbList>[]>(() => {
    const [status, available, notices] =
      getInstanceStatusColumns<EnhancedDbList>((row) => row.instance);

    return [
      status,
      available,
      {
        accessorKey: 'name',
        header: 'Name',
      },
      {
        id: 'technology',
        header: 'Technology',
        accessorFn: (row) => row.db?.spec.engine,
        Cell: ({ cell }) => {
          const engine = cell.getValue<Spec['engine']>();

          if (!engine) {
            return 'N/A';
          }

          return `${String(engine.type)} ${engine.version}`;
        },
      },
      {
        id: 'crd',
        accessorFn: (row) => row.db,
        header: 'CRD Version',
        Cell: ({ cell }) => {
          const db = cell.getValue<DbCluster | undefined>();

          return db?.status?.crVersion || 'N/A';
        },
      },
      {
        accessorKey: 'message',
        header: 'Actions',
        Cell: ({ cell, row }) => {
          const task = row.original.pendingTask;
          const message = cell.getValue<string>();

          if (task === 'restart' || task === 'upgradeEngine') {
            return (
              <Button
                data-testid="update-db-button"
                onClick={() => onDbClick(row.original)}
                sx={{
                  lineHeight: '20px',
                }}
              >
                {message}
              </Button>
            );
          }

          return message || task;
        },
      },
      notices,
    ];
  }, [dbClusters, onDbClick]);

  return (
    <>
      <Table
        getRowId={(row) => row.name}
        tableName={`${namespace}-upgrade-pending-actions`}
        noDataMessage="No pending actions"
        initialState={{ sorting: INSTANCE_STATUS_SORTING }}
        columns={columns}
        data={enhancedDbList}
        renderTopToolbarCustomActions={({ table }) => (
          <NeedsAttentionFilter
            table={table}
            count={countNeedingAttention(
              enhancedDbList.map((row) => row.instance)
            )}
          />
        )}
      />
      {selectedDbCluster.current && openUpdateCrDialog && (
        <UpdateCrDialog
          dbCluster={selectedDbCluster.current}
          onClose={() => setOpenUpdateCrDialog(false)}
        />
      )}
      {selectedDbCluster.current &&
        selectedDbEngineVersion.current &&
        openUpdateEngineDialog && (
          <UpdateEngineDialog
            dbCluster={selectedDbCluster.current}
            newVersion={selectedDbEngineVersion.current}
            onClose={() => setOpenUpdateEngineDialog(false)}
          />
        )}
    </>
  );
};

export default ClusterStatusTable;
