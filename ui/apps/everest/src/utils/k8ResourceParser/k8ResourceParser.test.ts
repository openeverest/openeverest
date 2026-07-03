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
  areResourceRequestsSynced,
  cpuParser,
  extractResourceCpuRequestValue,
  extractResourceCpuValue,
  extractResourceMemoryRequestValue,
  extractResourceMemoryValue,
  memoryParser,
} from '.';

describe('cpu parser', () => {
  // pattern is [description, input, output]
  const tests: [string, string, number][] = [
    ['parses full numbers', '1', 1],
    ['parses floats (< 1)', '1.5', 1.5],
    ['parses floats (> 1)', '0.5', 0.5],
    ['parses strings with milli (m) unit (whole number)', '1000m', 1],
    ['parses strings with milli (m) unit (decimal number)', '1300m', 1.3],
    ['parses strings with milli (m) unit (< 1)', '300m', 0.3],
  ];

  tests.map((t) =>
    it(`${t[0]} (${t[1]} to ${t[2]})`, () => {
      expect(cpuParser(t[1])).toEqual(t[2]);
    })
  );
});

describe('memory parser', () => {
  it('correctly parses memory strings', () => {
    expect(memoryParser('1')).toEqual({ value: 1, originalUnit: '' });
    expect(memoryParser('1k', 'G')).toEqual({
      value: 1 * 10 ** -6,
      originalUnit: 'k',
    });
    expect(memoryParser('1G', 'Gi')).toEqual({
      value: 10 ** 9 / 1024 ** 3,
      originalUnit: 'G',
    });
    expect(memoryParser('1G', 'G')).toEqual({ value: 1, originalUnit: 'G' });
  });
});

// Backward compatibility: older clusters stored cpu/memory as flat fields on
// `resources` (no limits/requests split). The extract helpers must keep reading
// those legacy payloads so the new UI renders old clusters correctly.
describe('resource value extraction (legacy vs new format)', () => {
  it('reads cpu/memory from new limits format', () => {
    const resources = {
      limits: { cpu: '2', memory: '4G' },
      requests: { cpu: '1', memory: '2G' },
    };

    expect(extractResourceCpuValue(resources)).toBe('2');
    expect(extractResourceMemoryValue(resources)).toBe('4G');
    expect(extractResourceCpuRequestValue(resources)).toBe('1');
    expect(extractResourceMemoryRequestValue(resources)).toBe('2G');
  });

  it('falls back to legacy flat cpu/memory fields', () => {
    const legacyResources = { cpu: '600m', memory: '1G' };

    expect(extractResourceCpuValue(legacyResources)).toBe('600m');
    expect(extractResourceMemoryValue(legacyResources)).toBe('1G');
    // Legacy clusters are treated as if limits and requests were equal, so the
    // flat value is returned as the effective request for every engine.
    expect(extractResourceCpuRequestValue(legacyResources)).toBe('600m');
    expect(extractResourceMemoryRequestValue(legacyResources)).toBe('1G');
  });

  it('always prefers explicit requests over the legacy flat fallback', () => {
    const resources = {
      limits: { cpu: '2', memory: '4G' },
      requests: { cpu: '1', memory: '2G' },
    };

    expect(extractResourceCpuRequestValue(resources)).toBe('1');
    expect(extractResourceMemoryRequestValue(resources)).toBe('2G');
  });

  it('prefers new limits over legacy flat fields when both are present', () => {
    const resources = {
      limits: { cpu: '2', memory: '4G' },
      cpu: '600m',
      memory: '1G',
    };

    expect(extractResourceCpuValue(resources)).toBe('2');
    expect(extractResourceMemoryValue(resources)).toBe('4G');
  });

  it('falls back to the limit (not the residual flat "0") when a synced legacy cluster was saved with limits only', () => {
    // After editing a legacy cluster with requests kept synced, only limits are
    // written and the flat cpu/memory fields are left as residual "0" values.
    // The effective request equals the limit, so the "0" must not leak through.
    const hybridResources = {
      cpu: '0',
      memory: '0',
      limits: { cpu: '1', memory: '2G' },
    };

    expect(extractResourceCpuRequestValue(hybridResources)).toBe('1');
    expect(extractResourceMemoryRequestValue(hybridResources)).toBe('2G');
  });

  it('returns 0 for missing cpu/memory', () => {
    expect(extractResourceCpuValue(undefined)).toBe(0);
    expect(extractResourceMemoryValue({})).toBe(0);
  });
});

describe('areResourceRequestsSynced', () => {
  it('treats legacy clusters (no requests) as synced', () => {
    expect(areResourceRequestsSynced({ cpu: '600m', memory: '1G' })).toBe(true);
  });

  it('treats absent requests as synced', () => {
    expect(
      areResourceRequestsSynced({ limits: { cpu: '2', memory: '4G' } })
    ).toBe(true);
  });

  it('treats a limits-only save with residual flat "0" fields as synced', () => {
    expect(
      areResourceRequestsSynced({
        cpu: '0',
        memory: '0',
        limits: { cpu: '1', memory: '2G' },
      })
    ).toBe(true);
  });

  it('treats equal requests and limits as synced', () => {
    expect(
      areResourceRequestsSynced({
        limits: { cpu: '2', memory: '4G' },
        requests: { cpu: '2', memory: '4G' },
      })
    ).toBe(true);
  });

  it('treats requests equal to limits across units as synced', () => {
    expect(
      areResourceRequestsSynced({
        limits: { cpu: '2000m', memory: '4G' },
        requests: { cpu: '2', memory: '4000M' },
      })
    ).toBe(true);
  });

  it('treats differing cpu requests as desynced', () => {
    expect(
      areResourceRequestsSynced({
        limits: { cpu: '2', memory: '4G' },
        requests: { cpu: '1', memory: '4G' },
      })
    ).toBe(false);
  });

  it('treats differing memory requests as desynced', () => {
    expect(
      areResourceRequestsSynced({
        limits: { cpu: '2', memory: '4G' },
        requests: { cpu: '2', memory: '2G' },
      })
    ).toBe(false);
  });
});
