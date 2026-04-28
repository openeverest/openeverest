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

import {
  PreviewContentText,
  TruncatedPreviewContentText,
} from '../preview-section';
import { AdvancedConfigurationType } from '../../database-form-schema.ts';
import { ProxyExposeType } from 'shared-types/dbCluster.types';
import { EMPTY_LOAD_BALANCER_CONFIGURATION } from 'consts.ts';

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
}: AdvancedConfigurationType) => {
  const isExternalAccessEnabled =
    exposureMethod === ProxyExposeType.LoadBalancer;

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
        <TruncatedPreviewContentText
          label="Database engine parameters set"
          text={engineParameters}
        />
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
