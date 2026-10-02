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

import { DbEngineType } from '@percona/types';

export type AffinityRule = {
  component: AffinityComponent;
  type: AffinityType;
  priority: AffinityPriority;
  weight?: number;
  topologyKey?: string;
  key?: string;
  operator?: AffinityOperator;
  values?: string;
  // uid is used to uniquely identify the rule on the client side
  uid: string;
};

type AffinityComponentType = keyof typeof AffinityComponent;

export type AffinityRules = [
  { component: AffinityComponentType; rules: AffinityRule[] },
];

export enum AffinityComponent {
  DbNode = 'engine',
  Proxy = 'proxy',
  ConfigServer = 'configServer',
}

export enum AffinityType {
  PodAntiAffinity = 'podAntiAffinity',
  PodAffinity = 'podAffinity',
  NodeAffinity = 'nodeAffinity',
}

export enum AffinityPriority {
  Preferred = 'preferred',
  Required = 'required',
}

export enum AffinityOperator {
  In = 'In',
  NotIn = 'NotIn',
  Exists = 'Exists',
  DoesNotExist = 'DoesNotExist',
  Gt = 'Gt',
  Lt = 'Lt',
}

// Label selectors (pod affinity) accept only these; node affinity adds Gt / Lt.
export const LABEL_SELECTOR_OPERATORS: AffinityOperator[] = [
  AffinityOperator.In,
  AffinityOperator.NotIn,
  AffinityOperator.Exists,
  AffinityOperator.DoesNotExist,
];

// Compare the node label value as an integer; take exactly one value.
export const NUMERIC_AFFINITY_OPERATORS: AffinityOperator[] = [
  AffinityOperator.Gt,
  AffinityOperator.Lt,
];

export const AffinityTypeValue: Record<AffinityType, string> = {
  [AffinityType.NodeAffinity]: 'Node affinity',
  [AffinityType.PodAffinity]: 'Pod affinity',
  [AffinityType.PodAntiAffinity]: 'Pod anti-affinity',
};

export const AffinityOperatorValue: Record<AffinityOperator, string> = {
  [AffinityOperator.Exists]: 'exists',
  [AffinityOperator.DoesNotExist]: 'does not exist',
  [AffinityOperator.In]: 'in',
  [AffinityOperator.NotIn]: 'not in',
  [AffinityOperator.Gt]: 'greater than',
  [AffinityOperator.Lt]: 'less than',
};

export const AffinityPriorityValue: Record<AffinityPriority, string> = {
  [AffinityPriority.Preferred]: 'Preferred',
  [AffinityPriority.Required]: 'Required',
};

export type AffinityMatchExpression = {
  key: string;
  operator: AffinityOperator;
  values?: string[];
};

type NodeAffinityPreference = {
  matchExpressions: AffinityMatchExpression[];
};

type NodeSelectorTerm = NodeAffinityPreference;

export type PodAffinityTerm = {
  labelSelector?: {
    matchExpressions: AffinityMatchExpression[];
  };
  topologyKey: string;
};

export type PreferredNodeSchedulingTerm = {
  preference: NodeAffinityPreference;
  weight: number;
};

export type PreferredPodSchedulingTerm = {
  weight: number;
  podAffinityTerm: PodAffinityTerm;
};

export type RequiredNodeSchedulingTerm = {
  nodeSelectorTerms: NodeSelectorTerm[];
};

export type RequiredPodSchedulingTerm = PodAffinityTerm[];

export type NodeAffinity = {
  preferredDuringSchedulingIgnoredDuringExecution?: PreferredNodeSchedulingTerm[];
  requiredDuringSchedulingIgnoredDuringExecution?: RequiredNodeSchedulingTerm;
};

export type PodAffinity = {
  preferredDuringSchedulingIgnoredDuringExecution?: PreferredPodSchedulingTerm[];
  requiredDuringSchedulingIgnoredDuringExecution?: RequiredPodSchedulingTerm;
};

export type PodAntiAffinity = PodAffinity;
export type Affinity = {
  nodeAffinity?: NodeAffinity;
} & {
  podAffinity?: PodAffinity;
} & {
  podAntiAffinity?: PodAntiAffinity;
};

export type PodSchedulingPolicyGetPayload = {
  items: PodSchedulingPolicy[];
};

export type PodSchedulingPolicy = {
  metadata: {
    name: string;
    finalizers: string[];
    generation: number;
    resourceVersion: string;
  };
  spec: {
    engineType: DbEngineType;
    affinityConfig: {
      [key in DbEngineType]?: {
        [key in AffinityComponent]?: Affinity;
      };
    };
  };
};
