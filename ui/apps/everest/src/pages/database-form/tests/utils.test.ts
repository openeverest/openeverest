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

import { formSubmitPostProcessing } from '../utils/form-submit-post-processing';
import {
  FieldType,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';

describe('formSubmitPostProcessing', () => {
  it('removes empty optional values recursively before submit payload is sent', () => {
    const input = {
      provider: 'psmdb',
      dbName: 'my-db',
      resources: {
        nodes: undefined,
        cpu: '',
        memory: null,
        disk: 10,
      },
      monitoring: {
        enabled: false,
      },
    } as Record<string, unknown>;

    const result = formSubmitPostProcessing({}, input);

    expect(result).toEqual({
      provider: 'psmdb',
      dbName: 'my-db',
      resources: {
        disk: 10,
      },
      monitoring: {
        enabled: false,
      },
    });
  });

  it('uses first multipath entry as source field and duplicates value to other target paths', () => {
    const schema: TopologyUISchemas = {
      replica: {
        sections: {
          resources: {
            components: {
              databaseVersion: {
                uiType: FieldType.Text,
                path: [
                  'spec.components.engine.version',
                  'spec.components.proxy.version',
                  'spec.components.configServer.version',
                ],
                fieldParams: {
                  label: 'Database version',
                },
              },
            },
          },
        },
      },
    };

    const input = {
      spec: {
        components: {
          engine: {
            version: '8.0.41',
          },
        },
      },
    } as Record<string, unknown>;

    const result = formSubmitPostProcessing({}, input, {
      schema,
      selectedTopology: 'replica',
    });

    expect(result).toEqual({
      spec: {
        components: {
          engine: {
            version: '8.0.41',
          },
          proxy: {
            version: '8.0.41',
          },
          configServer: {
            version: '8.0.41',
          },
        },
      },
    });
    // Payload should not include generated source field IDs.
    expect(
      (result as Record<string, unknown>)['g-databaseVersion']
    ).toBeUndefined();
  });
});
