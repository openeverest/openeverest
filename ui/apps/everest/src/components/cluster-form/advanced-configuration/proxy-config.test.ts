// everest
// Copyright (C) 2023 Percona LLC
// Copyright (C) 2026 The OpenEverest Contributors
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

import { describe, it, expect } from 'vitest';
import { DbType } from '@percona/types';
import { advancedConfigurationsSchema } from './advanced-configuration-schema';
import { advancedConfigurationModalDefaultValues } from './advanced-configuration.utils';
import { getProxyConfigLabel } from './messages';
import { changeDbClusterAdvancedConfig } from 'utils/db';
import { ProxyExposeType } from 'shared-types/dbCluster.types';
import { DbCluster } from 'shared-types/dbCluster.types';
import { DbEngineType } from '@percona/types';

const makeDbCluster = (overrides: Partial<DbCluster['spec']> = {}): DbCluster =>
  ({
    apiVersion: 'everest.percona.com/v1alpha1',
    kind: 'DatabaseCluster',
    metadata: { name: 'test', namespace: 'default' },
    spec: {
      engine: {
        type: DbEngineType.PXC,
        version: '8.0',
        replicas: 1,
        storage: { size: '1G', class: 'standard' },
        config: '',
      },
      proxy: {
        type: 'haproxy' as const,
        replicas: 1,
        expose: { type: ProxyExposeType.ClusterIP },
      },
      ...overrides,
    },
  }) as unknown as DbCluster;

describe('advancedConfigurationsSchema – proxyConfig fields', () => {
  const schema = advancedConfigurationsSchema();

  it('accepts proxyConfigEnabled=false with no proxyConfig', () => {
    const result = schema.safeParse({
      storageClass: 'standard',
      engineParametersEnabled: false,
      engineParameters: '',
      proxyConfigEnabled: false,
      proxyConfig: '',
      podSchedulingPolicyEnabled: false,
      podSchedulingPolicy: '',
      exposureMethod: ProxyExposeType.ClusterIP,
      loadBalancerConfigName: '',
      sourceRanges: [{ sourceRange: '' }],
      splitHorizonDNSEnabled: false,
      splitHorizonDNS: '',
    });
    expect(result.success).toBe(true);
  });

  it('accepts proxyConfigEnabled=true with a config string', () => {
    const result = schema.safeParse({
      storageClass: 'standard',
      engineParametersEnabled: false,
      engineParameters: '',
      proxyConfigEnabled: true,
      proxyConfig: 'max_connections=100',
      podSchedulingPolicyEnabled: false,
      podSchedulingPolicy: '',
      exposureMethod: ProxyExposeType.ClusterIP,
      loadBalancerConfigName: '',
      sourceRanges: [{ sourceRange: '' }],
      splitHorizonDNSEnabled: false,
      splitHorizonDNS: '',
    });
    expect(result.success).toBe(true);
  });

  it('rejects when proxyConfigEnabled is not a boolean', () => {
    const result = schema.safeParse({
      storageClass: 'standard',
      engineParametersEnabled: false,
      engineParameters: '',
      proxyConfigEnabled: 'yes',
      proxyConfig: '',
      podSchedulingPolicyEnabled: false,
      podSchedulingPolicy: '',
      exposureMethod: ProxyExposeType.ClusterIP,
      loadBalancerConfigName: '',
      sourceRanges: [{ sourceRange: '' }],
      splitHorizonDNSEnabled: false,
      splitHorizonDNS: '',
    });
    expect(result.success).toBe(false);
  });
});

describe('advancedConfigurationModalDefaultValues – proxyConfig read mapping', () => {
  it('sets proxyConfigEnabled=false and proxyConfig=undefined when spec.proxy.config is absent', () => {
    const cluster = makeDbCluster();
    const defaults = advancedConfigurationModalDefaultValues(cluster);
    expect(defaults.proxyConfigEnabled).toBe(false);
    expect(defaults.proxyConfig).toBeUndefined();
  });

  it('sets proxyConfigEnabled=true and proxyConfig when spec.proxy.config is set', () => {
    const cluster = makeDbCluster({
      proxy: {
        type: 'haproxy' as const,
        replicas: 1,
        expose: { type: ProxyExposeType.ClusterIP },
        config: 'max_connections=200',
      },
    });
    const defaults = advancedConfigurationModalDefaultValues(cluster);
    expect(defaults.proxyConfigEnabled).toBe(true);
    expect(defaults.proxyConfig).toBe('max_connections=200');
  });
});

describe('changeDbClusterAdvancedConfig – proxyConfig write mapping', () => {
  const base = makeDbCluster();

  it('sets spec.proxy.config when proxyConfigEnabled=true', () => {
    const result = changeDbClusterAdvancedConfig(
      base,
      false,
      ProxyExposeType.ClusterIP,
      '',
      [],
      false,
      '',
      '',
      '',
      true,
      'max_connections=100'
    );
    expect(result.spec.proxy.config).toBe('max_connections=100');
  });

  it('unsets spec.proxy.config when proxyConfigEnabled=true but proxyConfig is empty', () => {
    const result = changeDbClusterAdvancedConfig(
      base,
      false,
      ProxyExposeType.ClusterIP,
      '',
      [],
      false,
      '',
      '',
      '',
      true,
      ''
    );
    expect(result.spec.proxy.config).toBeUndefined();
  });

  it('unsets spec.proxy.config when proxyConfigEnabled=false', () => {
    const clusterWithConfig = makeDbCluster({
      proxy: {
        type: 'haproxy' as const,
        replicas: 1,
        expose: { type: ProxyExposeType.ClusterIP },
        config: 'max_connections=100',
      },
    });
    const result = changeDbClusterAdvancedConfig(
      clusterWithConfig,
      false,
      ProxyExposeType.ClusterIP,
      '',
      [],
      false,
      '',
      '',
      '',
      false,
      ''
    );
    expect(result.spec.proxy.config).toBeUndefined();
  });
});

describe('getProxyConfigLabel', () => {
  it('returns "Proxy Configuration" for PXC', () => {
    expect(getProxyConfigLabel(DbType.Mysql)).toBe('Proxy Configuration');
  });

  it('returns "Router Configuration" for PSMDB', () => {
    expect(getProxyConfigLabel(DbType.Mongo)).toBe('Router Configuration');
  });

  it('returns "PG Bouncer Configuration" for PostgreSQL', () => {
    expect(getProxyConfigLabel(DbType.Postresql)).toBe(
      'PG Bouncer Configuration'
    );
  });
});
