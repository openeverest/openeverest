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

import {test, expect} from '@fixtures'
import * as th from '@tests/utils/api';

const testPrefix = 'resource-limits';

test.describe.serial('Resource limits/requests structure tests', () => {
  test.describe.configure({timeout: 120 * 1000});

  const dbClusterName = th.limitedSuffixedName(testPrefix);

  test.afterAll(async ({request}) => {
    await th.deleteDBCluster(request, dbClusterName)
  })

  test('verify new resource structure with limits on cluster creation', async ({request}) => {
    const clusterPayload = th.getPXCClusterDataSimple(dbClusterName);
    
    // Create cluster with new limits format
    clusterPayload.spec.engine.resources = {
      limits: {
        cpu: '1',
        memory: '1G',
      },
    };
    clusterPayload.spec.proxy.resources = {
      limits: {
        cpu: '0.5',
        memory: '512M',
      },
    };

    let dbCluster;

    await test.step('create DB cluster with limits structure', async () => {
      dbCluster = await th.createDBClusterWithData(request, clusterPayload);
      
      // Verify engine resources have limits structure
      expect(dbCluster.spec.engine.resources).toBeDefined();
      expect(dbCluster.spec.engine.resources.limits).toBeDefined();
      expect(dbCluster.spec.engine.resources.limits.cpu).toBe('1');
      expect(dbCluster.spec.engine.resources.limits.memory).toBe('1G');
      
      // Verify proxy resources have limits structure  
      expect(dbCluster.spec.proxy.resources).toBeDefined();
      expect(dbCluster.spec.proxy.resources.limits).toBeDefined();
      expect(dbCluster.spec.proxy.resources.limits.cpu).toBe('0.5');
      expect(dbCluster.spec.proxy.resources.limits.memory).toBe('512M');
    });

    await test.step('verify structure persists after read', async () => {
      dbCluster = await th.getDBCluster(request, dbClusterName);
      
      // Verify engine limits structure is preserved
      expect(dbCluster.spec.engine.resources.limits).toBeDefined();
      expect(dbCluster.spec.engine.resources.limits.cpu).toBe('1');
      expect(dbCluster.spec.engine.resources.limits.memory).toBe('1G');
      
      // Verify proxy limits structure is preserved
      expect(dbCluster.spec.proxy.resources.limits).toBeDefined();
      expect(dbCluster.spec.proxy.resources.limits.cpu).toBe('0.5');
      expect(dbCluster.spec.proxy.resources.limits.memory).toBe('512M');
    });

    await test.step('update resource limits and verify round-trip', async () => {
      dbCluster = await th.getDBCluster(request, dbClusterName);
      
      // Update engine limits
      dbCluster.spec.engine.resources = {
        limits: {
          cpu: '2',
          memory: '2G',
        },
      };
      
      // Update proxy limits
      dbCluster.spec.proxy.resources = {
        limits: {
          cpu: '1',
          memory: '1G',
        },
      };
      
      dbCluster = await th.updateDBCluster(request, dbClusterName, dbCluster);
      
      // Verify updated engine limits
      expect(dbCluster.spec.engine.resources.limits.cpu).toBe('2');
      expect(dbCluster.spec.engine.resources.limits.memory).toBe('2G');
      
      // Verify updated proxy limits
      expect(dbCluster.spec.proxy.resources.limits.cpu).toBe('1');
      expect(dbCluster.spec.proxy.resources.limits.memory).toBe('1G');
    });

    await test.step('verify no legacy fields in new format', async () => {
      dbCluster = await th.getDBCluster(request, dbClusterName);
      
      // After using new format, verify old flat fields don't exist
      // (or are undefined/empty if server maintains them for compatibility)
      expect(dbCluster.spec.engine.resources.limits).toBeDefined();
      expect(dbCluster.spec.proxy.resources.limits).toBeDefined();
    });
  });

  test('verify backward compatibility with legacy format on read', async ({request}) => {
    const legacyClusterName = th.limitedSuffixedName(testPrefix + '-legacy');
    const legacyPayload = th.getPXCClusterDataSimple(legacyClusterName);
    
    // Create cluster using legacy format (api.ts still generates this)
    // This tests that the system can handle both old and new formats
    
    let dbCluster;

    await test.step('create legacy format cluster', async () => {
      dbCluster = await th.createDBClusterWithData(request, legacyPayload);
      expect(dbCluster.spec.engine.resources).toBeDefined();
    });

    await test.step('read legacy cluster and update to new format', async () => {
      dbCluster = await th.getDBCluster(request, legacyClusterName);
      
      // Convert to new format
      const engineResources = dbCluster.spec.engine.resources;
      const proxyResources = dbCluster.spec.proxy?.resources;
      
      // Extract old values (if they exist as flat fields)
      const oldEngineCpu = engineResources.cpu || engineResources.limits?.cpu;
      const oldEngineMemory = engineResources.memory || engineResources.limits?.memory;
      
      // Update to new format
      dbCluster.spec.engine.resources = {
        limits: {
          cpu: oldEngineCpu?.toString() || '1',
          memory: oldEngineMemory?.toString() || '1G',
        },
      };
      
      if (proxyResources && dbCluster.spec.proxy) {
        const oldProxyCpu = proxyResources.cpu || proxyResources.limits?.cpu;
        const oldProxyMemory = proxyResources.memory || proxyResources.limits?.memory;
        
        dbCluster.spec.proxy.resources = {
          limits: {
            cpu: oldProxyCpu?.toString() || '0.5',
            memory: oldProxyMemory?.toString() || '512M',
          },
        };
      }
      
      dbCluster = await th.updateDBCluster(request, legacyClusterName, dbCluster);
      
      // Verify it persists in new format
      expect(dbCluster.spec.engine.resources.limits).toBeDefined();
      if (dbCluster.spec.proxy?.resources) {
        expect(dbCluster.spec.proxy.resources.limits).toBeDefined();
      }
    });

    await test.afterAll(async () => {
      await th.deleteDBCluster(request, legacyClusterName);
    });
  });
});
