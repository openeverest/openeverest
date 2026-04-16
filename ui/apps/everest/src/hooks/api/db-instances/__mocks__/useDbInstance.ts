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

import { Instance } from 'shared-types/api.types';

export const DB_INSTANCE_QUERY_KEY = 'instance';

export const mockDbInstance: Instance = {
  apiVersion: 'core.openeverest.io/v1alpha1',
  kind: 'Instance',
  metadata: {
    name: 'pg-primary',
    namespace: 'default',
  } as unknown as Record<string, never>,
  spec: {
    provider: 'aws-provider',
    topology: { type: 'ha' },
  },
  status: { phase: 'Ready' },
};

export const useDbInstance = () => ({
  data: mockDbInstance,
  isLoading: false,
  isFetching: false,
  refetch: async () => ({ data: mockDbInstance }),
});
