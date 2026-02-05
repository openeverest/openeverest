// everest
// Copyright (C) 2023 Percona LLC
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

import { Stack } from '@mui/material';
import { DatabaseIcon, OverviewCard } from '@percona/ui-lib';
import { Messages } from '../../cluster-overview.messages';
import { DatabaseDetailsOverviewCardProps } from '../card.types';
import { BasicInformationSection } from './basic-information/basic';
import { ConnectionDetails } from './connection-details';
import { MonitoringDetails } from './monitoring/monitoring';
import { AdvancedConfiguration } from './advanced-configuration';
import { useMemo } from 'react';
import { useMonitoringInstancesForNamespace } from 'hooks';
import { useRBACPermissions } from 'hooks/rbac';

export const DbDetails = ({
  loading,
  type,
  name,
  namespace,
  version,
  loadingClusterDetails,
  port,
  username,
  password,
  connectionUrl,
  splitHorizonUrl,
  hostname,
  monitoring,
  externalAccess,
  parameters,
  storageClass,
  podSchedulingPolicy,
  loadBalancerConfig,
  exposeType,
  splitHorizonDNS,
  splitHorizonDomains,
}: DatabaseDetailsOverviewCardProps) => {
  const { data: monitoringInstances } =
    useMonitoringInstancesForNamespace(namespace);

  const monitoringInstancesToCheck = useMemo(
    () =>
      (monitoringInstances || []).map(
        (monitoringInstance) =>
          `${monitoringInstance.namespace}/${monitoringInstance.name}`
      ),
    [monitoringInstances]
  );

  const { canRead: canReadMonitoring } = useRBACPermissions(
    'monitoring-instances',
    monitoringInstancesToCheck.length > 0 ? monitoringInstancesToCheck : '*'
  );

  return (
    <OverviewCard
      dataTestId="database-details"
      cardHeaderProps={{
        title: Messages.titles.dbDetails,
        avatar: <DatabaseIcon />,
      }}
    >
      <Stack gap={3}>
        <BasicInformationSection
          loading={loading}
          type={type}
          name={name}
          namespace={namespace}
          version={version}
        />
        <ConnectionDetails
          clusterName={name}
          loading={loading}
          loadingClusterDetails={loadingClusterDetails}
          port={port}
          username={username}
          password={password}
          hostname={hostname}
          connectionUrl={connectionUrl}
          splitHorizonUrl={splitHorizonUrl}
          type={type}
          exposeType={exposeType}
        />
        {canReadMonitoring && (
          <MonitoringDetails loading={loading} monitoring={monitoring} />
        )}
        <AdvancedConfiguration
          externalAccess={externalAccess}
          parameters={parameters}
          storageClass={storageClass}
          podSchedulingPolicy={podSchedulingPolicy}
          loadBalancerConfig={loadBalancerConfig}
          splitHorizonDNS={splitHorizonDNS}
          splitHorizonDomains={splitHorizonDomains}
        />
      </Stack>
    </OverviewCard>
  );
};
