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

import { AffinityPriority, AffinityType } from 'shared-types/affinity.types';
import { AffinityGroup } from '../affinity-group.types';

export interface IndexedGroup {
  group: AffinityGroup;
  // Position in the stored list, used for edit / delete.
  index: number;
}

const TYPE_ORDER: Record<AffinityType, number> = {
  [AffinityType.NodeAffinity]: 0,
  [AffinityType.PodAffinity]: 1,
  [AffinityType.PodAntiAffinity]: 2,
};

const byType = (a: IndexedGroup, b: IndexedGroup) =>
  TYPE_ORDER[a.group.type] - TYPE_ORDER[b.group.type];

// Display order: hard constraints first (by type), then preferences by weight.
export const splitByPriority = (
  groups: AffinityGroup[]
): Record<AffinityPriority, IndexedGroup[]> => {
  const indexed = groups.map((group, index) => ({ group, index }));
  return {
    [AffinityPriority.Required]: indexed
      .filter(({ group }) => group.priority === AffinityPriority.Required)
      .sort(byType),
    [AffinityPriority.Preferred]: indexed
      .filter(({ group }) => group.priority === AffinityPriority.Preferred)
      .sort(
        (a, b) => (b.group.weight ?? 0) - (a.group.weight ?? 0) || byType(a, b)
      ),
  };
};

const isNodeRequired = ({ group }: IndexedGroup) =>
  group.type === AffinityType.NodeAffinity &&
  group.priority === AffinityPriority.Required;

// Required node-affinity groups are nodeSelectorTerms: a node needs to match
// only one of them, so adjacent ones form one OR run. Every other group stands
// alone (AND for required, by weight for preferred).
export const groupIntoOrRuns = (entries: IndexedGroup[]): IndexedGroup[][] =>
  entries.reduce<IndexedGroup[][]>((runs, entry) => {
    const last = runs[runs.length - 1];
    if (last && isNodeRequired(entry) && isNodeRequired(last[0])) {
      last.push(entry);
    } else {
      runs.push([entry]);
    }
    return runs;
  }, []);
