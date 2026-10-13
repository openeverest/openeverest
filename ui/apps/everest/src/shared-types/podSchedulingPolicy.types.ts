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

type InstanceComponent = NonNullable<
  NonNullable<Instance['spec']>['components']
>[string];

export type SchedulingPolicyType = keyof NonNullable<
  InstanceComponent['schedulingPolicy']
>;

// Runtime companion to SchedulingPolicyType. Keying by the union forces every
// policy type to be listed, so a contract change fails to compile here.
const schedulingPolicyTypeMap: Record<
  SchedulingPolicyType,
  SchedulingPolicyType
> = {
  affinity: 'affinity',
  nodeSelector: 'nodeSelector',
  tolerations: 'tolerations',
  schedulerName: 'schedulerName',
  topologySpreadConstraints: 'topologySpreadConstraints',
};

export const schedulingPolicyTypes: readonly SchedulingPolicyType[] =
  Object.values(schedulingPolicyTypeMap);

export type SchedulingSupportByComponent = Record<
  string,
  SchedulingPolicyType[]
>;
