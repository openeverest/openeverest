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
  AffinityOperator,
  AffinityPriority,
  AffinityType,
  NodeSelectorTerm,
  PodAffinityTerm,
} from 'shared-types/affinity.types';

// One expression. Conditions inside a group are AND-ed (one nodeSelectorTerm /
// one PodAffinityTerm's matchExpressions).
export interface AffinityCondition {
  key?: string;
  operator?: AffinityOperator;
  values?: string[];
}

// A group maps to a single term. For nodeAffinity the groups are OR-ed
// (nodeSelectorTerms); pod (anti)affinity has no native OR, so its groups become
// independent AND-ed PodAffinityTerms.
export interface AffinityGroup {
  type: AffinityType;
  priority: AffinityPriority;
  weight?: number; // preferred terms only
  topologyKey?: string; // pod (anti)affinity only
  conditions: AffinityCondition[];
  // The term as read. A save writes only the fields above into it, so what the
  // editor doesn't model (matchFields, matchLabels, namespaces, …) survives.
  source?: NodeSelectorTerm | PodAffinityTerm;
}
