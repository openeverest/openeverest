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

import { DbType } from '@percona/types';
import { DbCluster, ProxyExposeType } from 'shared-types/dbCluster.types';
import { AdvancedConfigurationFields } from './advanced-configuration.types';
import { AdvancedConfigurationFormType } from './advanced-configuration-schema';
import { EMPTY_LOAD_BALANCER_CONFIGURATION } from 'consts';

export const getParamsPlaceholderFromDbType = (dbType: DbType) => {
  let dynamicText = '';

  switch (dbType) {
    case DbType.Mongo:
      dynamicText = 'operationProfiling:\nmode: slowOp\nslowOpThresholdMs: 200';
      break;
    case DbType.Mysql:
      dynamicText =
        '[mysqld]\nkey_buffer_size=16M\nmax_allowed_packet=128M\nmax_connections=250';
      break;
    case DbType.Postresql:
    default:
      dynamicText =
        'log_connections = yes\nsearch_path = \'"$user", public\'\nshared_buffers = 128MB';
      break;
  }

  return dynamicText;
};

export const mapDeprecatedExposeType = (
  type: string | undefined
): ProxyExposeType => {
  if (type === 'internal') {
    return ProxyExposeType.ClusterIP;
  }
  if (type === 'external') {
    return ProxyExposeType.LoadBalancer;
  }
  if (
    type &&
    Object.values(ProxyExposeType).includes(type as ProxyExposeType)
  ) {
    return type as ProxyExposeType;
  }

  return ProxyExposeType.ClusterIP;
};

export const advancedConfigurationModalDefaultValues = (
  dbCluster: DbCluster
): AdvancedConfigurationFormType => {
  const sourceRangesSource = dbCluster?.spec?.proxy?.expose?.ipSourceRanges;
  const lbConfigName = dbCluster?.spec?.proxy?.expose?.loadBalancerConfigName;
  const exposeType = dbCluster?.spec?.proxy?.expose?.type;
  const mappedExposeType = mapDeprecatedExposeType(exposeType);

  return {
    [AdvancedConfigurationFields.storageClass]:
      dbCluster?.spec?.engine?.storage?.class || null,
    [AdvancedConfigurationFields.engineParametersEnabled]:
      !!dbCluster?.spec?.engine?.config,
    [AdvancedConfigurationFields.engineParameters]:
      dbCluster?.spec?.engine?.config,
    [AdvancedConfigurationFields.sourceRanges]: sourceRangesSource
      ? sourceRangesSource.map((sourceRange) => ({ sourceRange }))
      : [{ sourceRange: '' }],
    [AdvancedConfigurationFields.podSchedulingPolicyEnabled]:
      !!dbCluster?.spec.podSchedulingPolicyName,
    [AdvancedConfigurationFields.podSchedulingPolicy]:
      dbCluster?.spec.podSchedulingPolicyName,
    [AdvancedConfigurationFields.loadBalancerConfigName]:
      mappedExposeType === ProxyExposeType.LoadBalancer
        ? lbConfigName || EMPTY_LOAD_BALANCER_CONFIGURATION
        : '',
    [AdvancedConfigurationFields.exposureMethod]: mappedExposeType,
    [AdvancedConfigurationFields.splitHorizonDNSEnabled]:
      !!dbCluster?.spec.engineFeatures?.psmdb?.splitHorizonDnsConfigName,
    [AdvancedConfigurationFields.splitHorizonDNS]:
      dbCluster?.spec.engineFeatures?.psmdb?.splitHorizonDnsConfigName || '',
  };
};
