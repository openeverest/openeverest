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
  InstanceCondition,
  InstanceStatusSource,
} from 'shared-types/instance.types';

export const condition = (
  type: string,
  status: InstanceCondition['status'],
  reason = type
) => ({
  type,
  status,
  reason,
  message: '',
});

export const HEALTHY = [
  condition('Available', 'True'),
  condition('Ready', 'True'),
  condition('ConnectionDetailsReady', 'True'),
];

export const instance = (
  conditions: ReturnType<typeof condition>[],
  metadata: InstanceStatusSource['metadata'] = {}
): InstanceStatusSource => ({
  metadata: { generation: 2, ...metadata },
  status: { phase: 'Provisioned', observedGeneration: 2, conditions },
});
