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

import { ProxyExposeType } from 'shared-types/dbCluster.types';

export enum AdvancedConfigurationFields {
  sourceRanges = 'sourceRanges',
  engineParametersEnabled = 'engineParametersEnabled',
  engineParameters = 'engineParameters',
  storageClass = 'storageClass',
  podSchedulingPolicyEnabled = 'podSchedulingPolicyEnabled',
  podSchedulingPolicy = 'podSchedulingPolicy',
  exposureMethod = 'exposureMethod',
  loadBalancerConfigName = 'loadBalancerConfigName',
  splitHorizonDNS = 'splitHorizonDNS',
  splitHorizonDNSEnabled = 'splitHorizonDNSEnabled',
}

export const PROXY_EXPOSE_TYPE_TO_LABEL: Record<ProxyExposeType, string> = {
  [ProxyExposeType.ClusterIP]: 'Cluster IP',
  [ProxyExposeType.LoadBalancer]: 'Load balancer',
  [ProxyExposeType.NodePort]: 'Node port',
};

export type AllowedFieldsToInitiallyLoadDefaults =
  | 'storageClass'
  | 'podSchedulingPolicy'
  | 'loadBalancerConfigName'
  | 'splitHorizonDNS';
