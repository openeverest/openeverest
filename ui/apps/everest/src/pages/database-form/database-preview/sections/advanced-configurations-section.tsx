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

import { PreviewContentText } from '../preview-section';
import { AdvancedConfigurationType } from '../../database-form-schema.ts';
import { ProxyExposeType } from 'shared-types/dbCluster.types';
import { EMPTY_LOAD_BALANCER_CONFIGURATION } from 'consts.ts';
import { DbType } from '@percona/types';
import { getProxyConfigLabel } from 'components/cluster-form/advanced-configuration/messages';

type AdvancedConfigurationsPreviewProps = AdvancedConfigurationType & {
  dbType?: DbType;
  sharding?: boolean;
};

export const AdvancedConfigurationsPreviewSection = ({
  exposureMethod,
  loadBalancerConfigName,
  engineParametersEnabled,
  engineParameters,
  storageClass,
  podSchedulingPolicyEnabled,
  podSchedulingPolicy,
  splitHorizonDNSEnabled,
  splitHorizonDNS,
  proxyConfigEnabled,
  proxyConfig,
  dbType,
  sharding,
}: AdvancedConfigurationsPreviewProps) => {
  const isExternalAccessEnabled =
    exposureMethod === ProxyExposeType.LoadBalancer;
  const showProxyConfig = dbType !== DbType.Mongo || !!sharding;

  return (
    <>
      <PreviewContentText text={`Storage class: ${storageClass ?? ''}`} />
      <PreviewContentText
        text={`Ext. access: ${isExternalAccessEnabled ? 'enabled' : 'disabled'}`}
      />
      {isExternalAccessEnabled && (
        <PreviewContentText
          text={`Config name: ${loadBalancerConfigName ?? EMPTY_LOAD_BALANCER_CONFIGURATION}`}
        />
      )}
      {engineParametersEnabled && engineParameters && (
        <PreviewContentText text="Database engine parameters set" />
      )}
      {showProxyConfig && proxyConfigEnabled && proxyConfig && dbType && (
        <PreviewContentText text={`${getProxyConfigLabel(dbType)} set`} />
      )}
      {podSchedulingPolicyEnabled && podSchedulingPolicy && (
        <PreviewContentText
          text={`Pod scheduling policy: ${podSchedulingPolicy}`}
        />
      )}
      {splitHorizonDNSEnabled && splitHorizonDNS && (
        <PreviewContentText text={`Split-horizon DNS: ${splitHorizonDNS}`} />
      )}
    </>
  );
};
