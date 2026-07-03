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

import { useState } from 'react';
import { DatabaseIcon, OverviewCard } from '@percona/ui-lib';
import { Button, Stack } from '@mui/material';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import { SubmitHandler } from 'react-hook-form';
import { z } from 'zod';
import {
  CUSTOM_NR_UNITS_INPUT_VALUE,
  matchFieldsValueToResourceSize,
  NODES_DB_TYPE_MAP,
  NODES_DEFAULT_SIZES,
  PROXIES_DEFAULT_SIZES,
  resourcesFormSchema,
} from 'components/cluster-form';
import OverviewSection from '../overview-section';
import OverviewSectionRow from '../overview-section-row';
import { ResourcesDetailsOverviewProps } from './card.types';
import { Messages } from '../cluster-overview.messages';
import { ResourcesEditModal } from './resources';
import ResourcesSummary from './resources-summary';
import {
  cpuParser,
  getResourcesDetailedString,
  getTotalResourcesDetailedString,
  memoryParser,
  extractResourceCpuRequestValue,
  extractResourceCpuValue,
  extractResourceMemoryRequestValue,
  extractResourceMemoryValue,
  areResourceRequestsSynced,
} from 'utils/k8ResourceParser';
import { dbEngineToDbType } from '@percona/utils';
import { useUpdateDbClusterWithConflictRetry } from 'hooks';
import { DbType } from '@percona/types';
import {
  changeDbClusterResources,
  isProxy,
  getProxyUnitNamesFromDbType,
} from 'utils/db';
import { DbWizardFormFields } from 'consts';

export const ResourcesDetails = ({
  dbCluster,
  loading,
  sharding,
  canUpdate,
}: ResourcesDetailsOverviewProps) => {
  const [openEditModal, setOpenEditModal] = useState(false);
  const { mutate: updateCluster } = useUpdateDbClusterWithConflictRetry(
    dbCluster,
    {
      onSuccess: () => setOpenEditModal(false),
    }
  );
  const storageClass = dbCluster.spec.engine.storage.class;
  const cpu = extractResourceCpuValue(dbCluster.spec.engine.resources) || 0;
  const proxyCpu = isProxy(dbCluster.spec.proxy)
    ? extractResourceCpuValue(dbCluster.spec.proxy?.resources) || 0
    : 0;
  const memory =
    extractResourceMemoryValue(dbCluster.spec.engine.resources) || 0;
  const proxyMemory = isProxy(dbCluster.spec.proxy)
    ? extractResourceMemoryValue(dbCluster.spec.proxy?.resources) || 0
    : 0;
  const cpuRequests = extractResourceCpuRequestValue(
    dbCluster.spec.engine.resources
  );
  const proxyCpuRequests = isProxy(dbCluster.spec.proxy)
    ? extractResourceCpuRequestValue(dbCluster.spec.proxy?.resources)
    : undefined;
  const memoryRequests = extractResourceMemoryRequestValue(
    dbCluster.spec.engine.resources
  );
  const proxyMemoryRequests = isProxy(dbCluster.spec.proxy)
    ? extractResourceMemoryRequestValue(dbCluster.spec.proxy?.resources)
    : undefined;
  // Requests are considered "synced" with the limits when they are absent or
  // identical to the limits. When they differ, they were configured separately.
  const nodeRequestsSynced = areResourceRequestsSynced(
    dbCluster.spec.engine.resources
  );
  const proxyRequestsSynced = isProxy(dbCluster.spec.proxy)
    ? areResourceRequestsSynced(dbCluster.spec.proxy?.resources)
    : true;
  const disk = dbCluster.spec.engine.storage.size;
  const parsedDiskValues = memoryParser(disk.toString());
  const parsedMemoryValues = memoryParser(memory.toString(), 'G');
  const parsedProxyMemoryValues = memoryParser(proxyMemory.toString(), 'G');
  const dbType = dbEngineToDbType(dbCluster.spec.engine.type);
  const replicas = dbCluster.spec.engine.replicas.toString();
  const proxies = isProxy(dbCluster.spec.proxy)
    ? (dbCluster.spec.proxy.replicas || 0).toString()
    : '';
  const numberOfProxiesInt = proxies ? Math.floor(parseInt(proxies, 10)) : 0;
  const numberOfNodes = NODES_DB_TYPE_MAP[dbType].includes(replicas)
    ? replicas
    : CUSTOM_NR_UNITS_INPUT_VALUE;
  const numberOfNodesStr = replicas;
  const numberOfProxiesStr = NODES_DB_TYPE_MAP[dbType].includes(proxies)
    ? proxies
    : CUSTOM_NR_UNITS_INPUT_VALUE;

  const onSubmit: SubmitHandler<
    z.infer<ReturnType<typeof resourcesFormSchema>>
  > = ({
    cpu,
    disk,
    diskUnit,
    memory,
    cpuRequests,
    memoryRequests,
    proxyCpu,
    proxyMemory,
    proxyCpuRequests,
    proxyMemoryRequests,
    numberOfNodes,
    numberOfProxies,
    customNrOfNodes,
    customNrOfProxies,
    shardConfigServers,
    shardNr,
    nodeRequestsSynced: nodeRequestsSyncedInput,
    proxyRequestsSynced: proxyRequestsSyncedInput,
  }) => {
    updateCluster(
      changeDbClusterResources(
        dbCluster,
        {
          cpu,
          disk,
          diskUnit,
          memory,
          cpuRequests,
          memoryRequests,
          proxyCpu,
          proxyMemory,
          proxyCpuRequests,
          proxyMemoryRequests,
          nodeRequestsSynced: nodeRequestsSyncedInput,
          proxyRequestsSynced: proxyRequestsSyncedInput,
          numberOfProxies: parseInt(
            numberOfProxies === CUSTOM_NR_UNITS_INPUT_VALUE
              ? customNrOfProxies || '1'
              : numberOfProxies,
            10
          ),
          numberOfNodes: parseInt(
            numberOfNodes === CUSTOM_NR_UNITS_INPUT_VALUE
              ? customNrOfNodes || '1'
              : numberOfNodes,
            10
          ),
        },
        sharding?.enabled,
        shardNr,
        shardConfigServers
      )
    );
  };

  return (
    <>
      <OverviewCard
        dataTestId="resources"
        cardHeaderProps={{
          title: Messages.titles.resources,
          avatar: <DatabaseIcon />,
          ...{
            action: (
              <Button
                size="small"
                startIcon={<EditOutlinedIcon />}
                onClick={() => setOpenEditModal(true)}
                data-testid="edit-resources-button"
                disabled={!canUpdate}
              >
                Edit
              </Button>
            ),
          },
        }}
      >
        <Stack gap={3}>
          {dbType === DbType.Mongo && (
            <OverviewSection title={'Sharding'} loading={loading}>
              <OverviewSectionRow
                dataTestId="sharding-status"
                label={Messages.fields.status}
                content={
                  sharding?.enabled
                    ? Messages.fields.enabled
                    : Messages.fields.disabled
                }
              />
              {sharding?.enabled && (
                <OverviewSectionRow
                  dataTestId="number-of-shards"
                  label={Messages.fields.shards}
                  content={sharding?.shards?.toString()}
                />
              )}
              {sharding?.enabled && (
                <OverviewSectionRow
                  dataTestId="config-servers"
                  label={Messages.fields.configServers}
                  content={sharding?.configServer?.replicas?.toString()}
                />
              )}
            </OverviewSection>
          )}
          <OverviewSection
            title={`${numberOfNodesStr} node${+numberOfNodesStr > 1 ? 's' : ''} ${dbType === DbType.Mongo && sharding?.enabled ? 'per shard' : ''}`}
            loading={loading}
          >
            <ResourcesSummary
              synced={nodeRequestsSynced}
              rows={[
                {
                  label: Messages.fields.cpu,
                  dataTestId: 'node-cpu',
                  limit: getResourcesDetailedString(
                    cpuParser(cpu.toString() || '0'),
                    ''
                  ),
                  request:
                    cpuRequests !== undefined
                      ? getResourcesDetailedString(
                          cpuParser(cpuRequests.toString() || '0'),
                          ''
                        )
                      : undefined,
                },
                {
                  label: Messages.fields.memory,
                  limit: getResourcesDetailedString(
                    parsedMemoryValues.value,
                    'GB'
                  ),
                  request:
                    memoryRequests !== undefined
                      ? getResourcesDetailedString(
                          memoryParser(memoryRequests.toString(), 'G').value,
                          'GB'
                        )
                      : undefined,
                },
                {
                  label: Messages.fields.disk,
                  limit: getResourcesDetailedString(
                    parsedDiskValues.value,
                    parsedDiskValues.originalUnit
                  ),
                },
              ]}
            />
          </OverviewSection>
          {numberOfProxiesInt > 0 && (
            <OverviewSection
              title={`${proxies} ${getProxyUnitNamesFromDbType(dbEngineToDbType(dbCluster.spec.engine.type))[numberOfProxiesInt > 1 ? 'plural' : 'singular']}`}
              loading={loading}
            >
              <ResourcesSummary
                synced={proxyRequestsSynced}
                rows={[
                  {
                    label: Messages.fields.cpu,
                    dataTestId: `${getProxyUnitNamesFromDbType(dbEngineToDbType(dbCluster.spec.engine.type))[numberOfProxiesInt > 1 ? 'plural' : 'singular']}-cpu`,
                    limit: getTotalResourcesDetailedString(
                      cpuParser(proxyCpu.toString() || '0'),
                      parseInt(proxies, 10),
                      ''
                    ),
                    request:
                      proxyCpuRequests !== undefined
                        ? getTotalResourcesDetailedString(
                            cpuParser(proxyCpuRequests.toString() || '0'),
                            parseInt(proxies, 10),
                            ''
                          )
                        : undefined,
                  },
                  {
                    label: Messages.fields.memory,
                    limit: getTotalResourcesDetailedString(
                      parsedProxyMemoryValues.value,
                      parseInt(proxies, 10),
                      'GB'
                    ),
                    request:
                      proxyMemoryRequests !== undefined
                        ? getTotalResourcesDetailedString(
                            memoryParser(proxyMemoryRequests.toString(), 'G')
                              .value,
                            parseInt(proxies, 10),
                            'GB'
                          )
                        : undefined,
                  },
                ]}
              />
            </OverviewSection>
          )}
        </Stack>
      </OverviewCard>
      {openEditModal && (
        <ResourcesEditModal
          dbType={dbType}
          storageClass={storageClass || ''}
          allowDiskDescaling={false}
          shardingEnabled={!!sharding?.enabled}
          handleCloseModal={() => setOpenEditModal(false)}
          onSubmit={onSubmit}
          defaultValues={{
            dbType,
            [DbWizardFormFields.dbVersion]: dbCluster.spec.engine.version || '',
            cpu: cpuParser(cpu.toString() || '0'),
            cpuRequests: nodeRequestsSynced
              ? cpuParser(cpu.toString() || '0')
              : cpuParser((cpuRequests ?? cpu).toString() || '0'),
            disk: parsedDiskValues.value,
            diskUnit: parsedDiskValues.originalUnit,
            memory: memoryParser(memory.toString(), 'G').value,
            memoryRequests: nodeRequestsSynced
              ? memoryParser(memory.toString(), 'G').value
              : memoryParser((memoryRequests ?? memory).toString(), 'G').value,
            proxyCpu: cpuParser(proxyCpu.toString() || '0'),
            proxyCpuRequests: proxyRequestsSynced
              ? cpuParser(proxyCpu.toString() || '0')
              : cpuParser((proxyCpuRequests ?? proxyCpu).toString() || '0'),
            proxyMemory: memoryParser(proxyMemory.toString(), 'G').value,
            proxyMemoryRequests: proxyRequestsSynced
              ? memoryParser(proxyMemory.toString(), 'G').value
              : memoryParser(
                  (proxyMemoryRequests ?? proxyMemory).toString(),
                  'G'
                ).value,
            [DbWizardFormFields.nodeRequestsSynced]: nodeRequestsSynced,
            [DbWizardFormFields.proxyRequestsSynced]: proxyRequestsSynced,
            sharding: !!sharding?.enabled,
            ...(!!sharding?.enabled && {
              shardConfigServers: sharding?.configServer?.replicas,
              shardNr: sharding?.shards.toString(),
            }),
            numberOfNodes: numberOfNodes,
            numberOfProxies: numberOfProxiesStr,
            customNrOfNodes: replicas,
            customNrOfProxies: proxies,
            resourceSizePerNode: matchFieldsValueToResourceSize(
              NODES_DEFAULT_SIZES(dbType, dbCluster.spec.engine.version),
              dbCluster.spec.engine.resources
            ),
            resourceSizePerProxy: matchFieldsValueToResourceSize(
              PROXIES_DEFAULT_SIZES[dbType],
              isProxy(dbCluster.spec.proxy)
                ? dbCluster.spec.proxy.resources
                : undefined
            ),
          }}
        />
      )}
    </>
  );
};
