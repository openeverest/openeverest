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

import { describe, it, expect } from 'vitest';
import { DbType } from '@percona/types';
import { ProxyExposeType } from 'shared-types/dbCluster.types';
import { getProxySpec } from './utils';

describe('getProxySpec', () => {
  it('writes both limits and requests by default', () => {
    const proxy = getProxySpec(
      DbType.Mysql,
      '1',
      '',
      ProxyExposeType.ClusterIP,
      1,
      2,
      false
    );

    expect(proxy.resources?.limits).toEqual({ cpu: '1', memory: '2G' });
    expect(proxy.resources?.requests).toEqual({ cpu: '1', memory: '2G' });
  });

  it('uses explicit request values when provided', () => {
    const proxy = getProxySpec(
      DbType.Mysql,
      '1',
      '',
      ProxyExposeType.ClusterIP,
      2,
      4,
      false,
      1,
      2
    );

    expect(proxy.resources?.limits).toEqual({ cpu: '2', memory: '4G' });
    expect(proxy.resources?.requests).toEqual({ cpu: '1', memory: '2G' });
  });

  it('omits requests when omitRequests is true', () => {
    const proxy = getProxySpec(
      DbType.Mysql,
      '1',
      '',
      ProxyExposeType.ClusterIP,
      1,
      2,
      false,
      undefined,
      undefined,
      undefined,
      undefined,
      false,
      '',
      true
    );

    expect(proxy.resources?.limits).toEqual({ cpu: '1', memory: '2G' });
    expect(proxy.resources?.requests).toBeUndefined();
  });

  it('returns only the expose config for a non-sharded MongoDB cluster', () => {
    const proxy = getProxySpec(
      DbType.Mongo,
      '1',
      '',
      ProxyExposeType.ClusterIP,
      1,
      2,
      false
    );

    expect(proxy.resources).toBeUndefined();
    expect(proxy.expose?.type).toBe(ProxyExposeType.ClusterIP);
  });
});
