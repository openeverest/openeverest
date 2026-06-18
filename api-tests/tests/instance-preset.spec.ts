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
import {EVEREST_CI_NAMESPACE, TIMEOUTS} from '@root/constants';
import {checkError} from '@tests/utils/api';

const PROVIDER_NAME = 'test-provider';
const PRESET_NAME = 'test-preset';
const INSTANCE_NAME = 'test-instance-from-preset';
const CLUSTER_NAME = 'main';

test.describe('Instance Preset tests', () => {
  test.describe.configure({timeout: TIMEOUTS.OneMinute});

  test.afterAll(async ({request}) => {
    // Clean up instance
    try {
      await request.delete(`/v1/clusters/${CLUSTER_NAME}/namespaces/${EVEREST_CI_NAMESPACE}/instances/${INSTANCE_NAME}`);
      console.log('Instance deleted successfully');
    } catch (error) {
      console.error('Failed to delete instance:', error);
    }
  });

  test('list instance presets', async ({request}) => {
    const response = await request.get(
      `/v1/clusters/${CLUSTER_NAME}/instance-presets`
    );

    await checkError(response);
    const presetList = await response.json();
    
    expect(presetList.items).toBeTruthy();
    const foundPreset = presetList.items.find((p: any) => p.metadata.name === PRESET_NAME);
    expect(foundPreset).toBeTruthy();
    expect(foundPreset.spec.provider).toBe(PROVIDER_NAME);
    expect(foundPreset.spec.version).toBe('1.0.0');
  });

  test('get specific instance preset', async ({request}) => {
    const response = await request.get(
      `/v1/clusters/${CLUSTER_NAME}/instance-presets/${PRESET_NAME}`
    );

    await checkError(response);
    const preset = await response.json();
    
    expect(preset.metadata.name).toBe(PRESET_NAME);
    expect(preset.spec.provider).toBe(PROVIDER_NAME);
    expect(preset.spec.version).toBe('1.0.0');
    expect(preset.spec).toBeTruthy();
    // Verify namespace scope secret is empty
    expect(preset.spec.components.engine.config.secretRef.name).toBeUndefined();
  });
  
  test('create instance using preset', async ({request}) => {
    await test.step('create instance from preset', async () => {
      // First, resolve the preset to get the complete instance spec
      const resolveResponse = await request.get(
        `/v1/clusters/${CLUSTER_NAME}/instance-presets/${PRESET_NAME}/resolve?namespace=${EVEREST_CI_NAMESPACE}`
      );

      await checkError(resolveResponse);
      const preset = await resolveResponse.json();

      // Verify namespace scope secret has default filled fields
      expect(preset.spec.components.engine.config.secretRef.name).toBe("test-secret");

      // Copy the preset spec and add annotation
      const instancePayload = {
        metadata: {
          name: INSTANCE_NAME,
          annotations: {
            'openeverest.io/instance-preset': PRESET_NAME,
          },
        },
        spec: preset.spec,
      };

      // Use the spec from preset to create the instance
      const response = await request.post(
        `/v1/clusters/${CLUSTER_NAME}/namespaces/${EVEREST_CI_NAMESPACE}/instances`,
        {
          data: instancePayload,
        }
      );

      await checkError(response);
    });

    await test.step('verify instance was created', async () => {
      await expect(async () => {
        const response = await request.get(
          `/v1/clusters/${CLUSTER_NAME}/namespaces/${EVEREST_CI_NAMESPACE}/instances/${INSTANCE_NAME}`
        );
        
        await checkError(response);
        const instance = await response.json();
        
        expect(instance.metadata.name).toBe(INSTANCE_NAME);
        expect(instance.spec.components.engine.config.secretRef.name).toBe("test-secret");
      }).toPass({
        intervals: [2000],
        timeout: TIMEOUTS.OneMinute,
      });
    });
  });
});
