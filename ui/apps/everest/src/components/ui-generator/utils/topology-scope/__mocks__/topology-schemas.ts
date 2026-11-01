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
  Component,
  FieldType,
  TopologyUISchemas,
} from '../../../ui-generator.types';

export const numberField = (path: string): Component => ({
  uiType: FieldType.Number,
  path,
  fieldParams: { label: path },
});

const monitoringField: Component = {
  uiType: FieldType.Toggle,
  path: 'spec.components.monitoring.enabled',
  fieldParams: { label: 'Monitoring' },
};

// Milvus-shaped: each topology owns its own components; monitoring is shared.
export const twoTopologySchema: TopologyUISchemas = {
  standalone: {
    sections: {
      resources: {
        components: {
          replicas: numberField('spec.components.standalone.replicas'),
          monitoring: monitoringField,
        },
      },
    },
  },
  cluster: {
    sections: {
      resources: {
        components: {
          coordinator: numberField('spec.components.mixCoord.replicas'),
          proxy: numberField('spec.components.proxy.replicas'),
          monitoring: monitoringField,
        },
      },
    },
  },
};
