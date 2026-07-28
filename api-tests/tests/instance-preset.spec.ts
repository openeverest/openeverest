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

import {test, expect} from '@fixtures';
import {EVEREST_CI_NAMESPACE, TIMEOUTS} from '@root/constants';
import {checkError} from '@tests/utils/api';

const PROVIDER_NAME = 'test-provider';
const PRESET_NAME = 'test-preset';
const CRUD_PRESET_NAME = 'test-preset-crud';
const FROM_INSTANCE_PRESET_NAME = 'test-preset-from-instance';
const INSTANCE_NAME = 'test-instance-from-preset';
const SOURCE_INSTANCE_NAME = 'test-source-instance';
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

    // Clean up source instance
    try {
      await request.delete(`/v1/clusters/${CLUSTER_NAME}/namespaces/${EVEREST_CI_NAMESPACE}/instances/${SOURCE_INSTANCE_NAME}`);
      console.log('Source instance deleted successfully');
    } catch (error) {
      console.error('Failed to delete source instance:', error);
    }

    // Clean up preset created from instance
    try {
      await request.delete(`/v1/clusters/${CLUSTER_NAME}/instance-presets/${FROM_INSTANCE_PRESET_NAME}`);
      console.log('Preset from instance deleted successfully');
    } catch (error) {
      console.error('Failed to delete preset from instance:', error);
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
    expect(foundPreset.spec.providerRef.name).toBe(PROVIDER_NAME);
    expect(foundPreset.spec.version).toBe('1.0.0');
  });

  test('get specific instance preset', async ({request}) => {
    const response = await request.get(
      `/v1/clusters/${CLUSTER_NAME}/instance-presets/${PRESET_NAME}`
    );

    await checkError(response);
    const preset = await response.json();

    expect(preset.metadata.name).toBe(PRESET_NAME);
    expect(preset.spec.providerRef.name).toBe(PROVIDER_NAME);
    expect(preset.spec.version).toBe('1.0.0');
    expect(preset.spec).toBeTruthy();
    // Inline engine configuration is carried verbatim on the preset.
    expect(preset.spec.components.engine.parameters.configuration).toBe(
      'key = value\n'
    );
  });
  
  test('create instance using preset', async ({request}) => {
    await test.step('create instance from preset', async () => {
      // First, resolve the preset to get the complete instance spec
      const resolveResponse = await request.get(
        `/v1/clusters/${CLUSTER_NAME}/instance-presets/${PRESET_NAME}/resolve?namespace=${EVEREST_CI_NAMESPACE}`
      );

      await checkError(resolveResponse);
      const preset = await resolveResponse.json();

      // Inline parameters are untouched by resolution; storage defaults are filled.
      expect(preset.spec.components.engine.parameters.configuration).toBe(
        'key = value\n'
      );
      expect(preset.spec.components.engine.storage.storageClass).toBe("local-path");

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
        expect(instance.spec.components.engine.parameters.configuration).toBe(
          'key = value\n'
        );
      }).toPass({
        intervals: [2000],
        timeout: TIMEOUTS.OneMinute,
      });
    });
  });

  test('create/update/delete instance preset', async ({request}) => {
    const presetPayload = {
      metadata: {
        name: CRUD_PRESET_NAME,
      },
      spec: {
        provider: PROVIDER_NAME,
        version: '1.0.0',
        components: {
          engine: {
            type: 'test',
            replicas: 3,
            storage: {
              size: '10Gi',
              storageClass: 'local-path',
            },
            resources: {
              limits: {
                cpu: '1',
                memory: '2Gi',
              },
            },
            customSpec: {
              enableUnsafeMode: true,
            },
          },
        },
      },
    };

    const response = await request.post(
      `/v1/clusters/${CLUSTER_NAME}/instance-presets`,
      {
        data: presetPayload,
      }
    );

    await checkError(response);
    expect(response.status()).toBe(201);

    await test.step('verify preset was created', async () => {
      const response = await request.get(
        `/v1/clusters/${CLUSTER_NAME}/instance-presets/${CRUD_PRESET_NAME}`
      );

      await checkError(response);
      const preset = await response.json();

      expect(preset.metadata.name).toBe(CRUD_PRESET_NAME);
      expect(preset.spec.provider).toBe(PROVIDER_NAME);
      expect(preset.spec.version).toBe('1.0.0');
      expect(preset.spec.components.engine.type).toBe('test');
      expect(preset.spec.components.engine.replicas).toBe(3);
      expect(preset.spec.components.engine.storage.size).toBe('10Gi');
      expect(preset.spec.components.engine.storage.storageClass).toBe('local-path');
      expect(preset.spec.components.engine.resources.limits.cpu).toBe('1');
      expect(preset.spec.components.engine.resources.limits.memory).toBe('2Gi');
      expect(preset.spec.components.engine.customSpec.enableUnsafeMode).toBe(true);
    });

    await test.step('update preset spec', async () => {
      // Get current preset to use as base for update
      const getResponse = await request.get(
        `/v1/clusters/${CLUSTER_NAME}/instance-presets/${CRUD_PRESET_NAME}`
      );

      await checkError(getResponse);
      const preset = await getResponse.json();
      expect(preset.metadata.name).toBe(CRUD_PRESET_NAME);

      // Build update payload based on current preset
      const updatedPayload = {
        ...preset,
        spec: {
          ...preset.spec,
          version: '2.0.0',
          components: {
            ...preset.spec.components,
            engine: {
              ...preset.spec.components.engine,
              replicas: 5, // Changed from 3 to 5
              storage: {
                ...preset.spec.components.engine.storage,
                size: '20Gi', // Changed from 10Gi to 20Gi
              },
              resources: {
                limits: {
                  cpu: '2', // Changed from 1 to 2
                  memory: '4Gi', // Changed from 2Gi to 4Gi
                },
              },
            },
          },
        },
      };

      const response = await request.put(
        `/v1/clusters/${CLUSTER_NAME}/instance-presets/${CRUD_PRESET_NAME}`,
        {
          data: updatedPayload,
        }
      );

      await checkError(response);
      expect(response.status()).toBe(200);

      await test.step('verify preset was updated', async () => {
        const response = await request.get(
          `/v1/clusters/${CLUSTER_NAME}/instance-presets/${CRUD_PRESET_NAME}`
        );

        await checkError(response);
        const preset = await response.json();

        expect(preset.metadata.name).toBe(CRUD_PRESET_NAME);
        expect(preset.spec.provider).toBe(PROVIDER_NAME);
        expect(preset.spec.version).toBe('2.0.0');
        expect(preset.spec.components.engine.type).toBe('test');
        expect(preset.spec.components.engine.replicas).toBe(5);
        expect(preset.spec.components.engine.storage.size).toBe('20Gi');
        expect(preset.spec.components.engine.storage.storageClass).toBe(
          'local-path'
        );
        expect(preset.spec.components.engine.resources.limits.cpu).toBe('2');
        expect(preset.spec.components.engine.resources.limits.memory).toBe(
          '4Gi'
        );
        expect(
          preset.spec.components.engine.customSpec.enableUnsafeMode
        ).toBe(true);
      });
    });

    await test.step('delete preset', async () => {
      const response = await request.delete(
        `/v1/clusters/${CLUSTER_NAME}/instance-presets/${CRUD_PRESET_NAME}`
      );

      await checkError(response);
      expect(response.status()).toBe(204);
    });

    await test.step('verify preset is deleted', async () => {
      const response = await request.get(
        `/v1/clusters/${CLUSTER_NAME}/instance-presets/${CRUD_PRESET_NAME}`
      );

      expect(response.status()).toBe(404);
    });
  });

  test('create preset from instance', async ({request}) => {
    await test.step('create source instance', async () => {
      // Create an instance with a full spec that will be used as source for preset creation
      const instancePayload = {
        metadata: {
          name: SOURCE_INSTANCE_NAME,
        },
        spec: {
          provider: PROVIDER_NAME,
          version: '1.0.0',
          components: {
            engine: {
              type: 'test',
              replicas: 3,
              storage: {
                size: '15Gi',
                storageClass: 'local-path',
              },
              resources: {
                limits: {
                  cpu: '1',
                  memory: '2Gi',
                },
              },
              config: {
                configMapRef: {
                  name: 'test-configmap',
                },
              },
            },
          },
        },
      };

      const response = await request.post(
        `/v1/clusters/${CLUSTER_NAME}/namespaces/${EVEREST_CI_NAMESPACE}/instances`,
        {
          data: instancePayload,
        }
      );

      await checkError(response);
      expect(response.status()).toBe(201);
    });

    await test.step('create preset from instance', async () => {
      const response = await request.post(
        `/v1/clusters/${CLUSTER_NAME}/instance-presets/from-instance`,
        {
          data: {
            name: FROM_INSTANCE_PRESET_NAME,
            instanceName: SOURCE_INSTANCE_NAME,
            instanceNamespace: EVEREST_CI_NAMESPACE,
          },
        }
      );

      await checkError(response);
      expect(response.status()).toBe(201);
    });

    await test.step('verify preset was created', async () => {
      const response = await request.get(
        `/v1/clusters/${CLUSTER_NAME}/instance-presets/${FROM_INSTANCE_PRESET_NAME}`
      );

      await checkError(response);
      const preset = await response.json();

      expect(preset.metadata.name).toBe(FROM_INSTANCE_PRESET_NAME);
      expect(preset.spec.provider).toBe(PROVIDER_NAME);
      expect(preset.spec.version).toBe('1.0.0');
      expect(preset.spec.components.engine.type).toBe('test');
      expect(preset.spec.components.engine.replicas).toBe(3);
      expect(preset.spec.components.engine.resources.limits.cpu).toBe('1');
      expect(preset.spec.components.engine.resources.limits.memory).toBe('2Gi');
      expect(preset.spec.components.engine.storage.storageClass).toBe('local-path');
      expect(preset.spec.components.engine.storage.size).toBe('15Gi');
      // Verify namespace-scoped fields are cleared
      expect(preset.spec.components.engine.config.configMapRef.name).toBeUndefined();
    });
  });
});
