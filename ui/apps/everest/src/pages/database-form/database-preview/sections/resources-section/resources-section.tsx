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

import { CUSTOM_NR_UNITS_INPUT_VALUE } from 'components/cluster-form';
import { getProxyUnitNamesFromDbType } from 'utils/db';
import { DbType } from '@percona/types';
import { SectionProps } from '../section.types';
import { ResourceGroup, ResourceTableRow } from './resource-group';
import { SectionHeading } from './section-heading';
import { formatResourceValue } from './resources-section.utils';

export const ResourcesPreviewSection = ({
  dbType,
  numberOfNodes,
  customNrOfNodes,
  numberOfProxies,
  customNrOfProxies,
  cpu,
  disk,
  diskUnit,
  memory,
  proxyCpu,
  proxyMemory,
  cpuRequests,
  memoryRequests,
  proxyCpuRequests,
  proxyMemoryRequests,
  nodeRequestsSynced,
  proxyRequestsSynced,
  sharding,
  shardNr,
  shardConfigServers,
}: SectionProps) => {
  const proxyUnitNames = getProxyUnitNamesFromDbType(dbType);
  if (numberOfNodes === CUSTOM_NR_UNITS_INPUT_VALUE) {
    numberOfNodes = customNrOfNodes || '';
  }

  if (numberOfProxies === CUSTOM_NR_UNITS_INPUT_VALUE) {
    numberOfProxies = customNrOfProxies || '';
  }

  let intNumberOfNodes = Math.max(parseInt(numberOfNodes, 10), 0);
  let intNumberOfProxies = Math.max(parseInt(numberOfProxies, 10), 0);

  if (Number.isNaN(intNumberOfNodes)) {
    intNumberOfNodes = 0;
  }

  if (Number.isNaN(intNumberOfProxies)) {
    intNumberOfProxies = 0;
  }

  const parsedCPU = Number(cpu) * intNumberOfNodes;
  const parsedDisk = Number(disk) * intNumberOfNodes;
  const parsedMemory = Number(memory) * intNumberOfNodes;
  const parsedProxyCPU = Number(proxyCpu) * intNumberOfProxies;
  const parsedProxyMemory = Number(proxyMemory) * intNumberOfProxies;
  const parsedShardNr = shardNr ? Number.parseInt(shardNr) : undefined;
  const nodesTotalNumber =
    sharding && shardNr ? +shardNr * intNumberOfNodes : intNumberOfNodes;

  const nodesText = `${nodesTotalNumber} ${nodesTotalNumber === 1 ? 'node' : 'nodes'}`;
  const proxyText = `${intNumberOfProxies} ${intNumberOfProxies === 1 ? proxyUnitNames.singular : proxyUnitNames.plural}`;

  const parsedCpuRequests = Number(cpuRequests) * intNumberOfNodes;
  const parsedMemoryRequests = Number(memoryRequests) * intNumberOfNodes;
  const parsedProxyCpuRequests = Number(proxyCpuRequests) * intNumberOfProxies;
  const parsedProxyMemoryRequests =
    Number(proxyMemoryRequests) * intNumberOfProxies;

  const nodesShardMultiplier =
    sharding && parsedShardNr ? parsedShardNr : undefined;

  const nodeRows: ResourceTableRow[] = [
    {
      label: 'CPU',
      limit: formatResourceValue(parsedCPU, 'CPU', nodesShardMultiplier),
      request: formatResourceValue(
        parsedCpuRequests,
        'CPU',
        nodesShardMultiplier
      ),
    },
    {
      label: 'Memory',
      limit: formatResourceValue(parsedMemory, 'GB', nodesShardMultiplier),
      request: formatResourceValue(
        parsedMemoryRequests,
        'GB',
        nodesShardMultiplier
      ),
    },
    {
      label: 'Disk',
      limit: formatResourceValue(parsedDisk, diskUnit, nodesShardMultiplier),
      request: undefined,
    },
  ];

  const proxyRows: ResourceTableRow[] = [
    {
      label: 'CPU',
      limit: formatResourceValue(parsedProxyCPU, 'CPU'),
      request: formatResourceValue(parsedProxyCpuRequests, 'CPU'),
    },
    {
      label: 'Memory',
      limit: formatResourceValue(parsedProxyMemory, 'GB'),
      request: formatResourceValue(parsedProxyMemoryRequests, 'GB'),
    },
  ];

  return (
    <>
      {sharding && <SectionHeading>{`${shardNr} shards`}</SectionHeading>}
      <ResourceGroup
        title={nodesText}
        rows={nodeRows}
        synced={nodeRequestsSynced !== false}
        dataTestId="nodes-resources-table"
      />
      {(dbType !== DbType.Mongo || sharding) && (
        <ResourceGroup
          title={proxyText}
          rows={proxyRows}
          synced={proxyRequestsSynced !== false}
          dataTestId="proxies-resources-table"
        />
      )}
      {sharding && (
        <SectionHeading>{`${shardConfigServers} configuration servers`}</SectionHeading>
      )}
    </>
  );
};
