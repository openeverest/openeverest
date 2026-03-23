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

import { Instance, InstanceConnectionDetails } from 'types/api';
import { DbInstanceForNamespaceResult } from '../useDbInstanceList';

export const mockInstances: Instance[] = [
  {
    apiVersion: 'core.openeverest.io/v1alpha1',
    kind: 'Instance',
    metadata: {},
    spec: {
      provider: 'aws-provider',
      components: {
        postgresql: {
          replicas: 3,
          resources: {
            requests: { cpu: '500m', memory: '1Gi' },
            limits: { cpu: '1', memory: '2Gi' },
          },
          storage: {
            size: '10Gi',
            storageClass: 'gp3',
          },
          version: '16.2',
          type: 'postgresql',
        },
      },
      topology: {
        type: 'ha',
      },
    },
    status: {
      phase: 'ready',
      components: [
        {
          pods: [{ name: 'pg-instance-1-0' }],
          ready: 3,
          state: 'ready',
          total: 3,
        },
      ],
      conditions: [
        {
          lastTransitionTime: '2026-03-20T10:00:00Z',
          message: 'Instance is ready',
          reason: 'Ready',
          status: 'True',
          type: 'Ready',
        },
      ],
      connectionSecretRef: { name: 'pg-instance-1-connection' },
    },
  },
  {
    apiVersion: 'core.openeverest.io/v1alpha1',
    kind: 'Instance',
    metadata: {},
    spec: {
      provider: 'gcp-provider',
      components: {
        mongodb: {
          replicas: 1,
          resources: {
            requests: { cpu: '250m', memory: '512Mi' },
          },
          storage: {
            size: '5Gi',
          },
          version: '7.0',
          type: 'mongodb',
        },
      },
      topology: {
        type: 'standalone',
      },
    },
    status: {
      phase: 'creating',
      components: [],
      conditions: [],
    },
  },
];

export const mockInstanceConnection: InstanceConnectionDetails = {
  host: 'pg-instance-1.ns-1.svc.cluster.local',
  port: '5432',
  username: 'admin',
  password: 'secret-password',
  uri: 'postgresql://admin:secret-password@pg-instance-1.ns-1.svc.cluster.local:5432/defaultdb',
  provider: 'aws-provider',
  type: 'postgresql',
};

export const useDbInstanceList = () => ({
  data: mockInstances,
  isLoading: false,
  isFetching: false,
});

export const useInstancesForNamespaces = (): DbInstanceForNamespaceResult[] => [
  {
    namespace: 'ns-1',
    queryResult: {
      data: [mockInstances[0]],
      isSuccess: true,
      isLoading: false,
      isFetching: false,
    } as DbInstanceForNamespaceResult['queryResult'],
  },
  {
    namespace: 'ns-2',
    queryResult: {
      data: [mockInstances[1]],
      isSuccess: true,
      isLoading: false,
      isFetching: false,
    } as DbInstanceForNamespaceResult['queryResult'],
  },
];

export const useInstanceConnection = () => ({
  data: mockInstanceConnection,
  isLoading: false,
  isFetching: false,
});
