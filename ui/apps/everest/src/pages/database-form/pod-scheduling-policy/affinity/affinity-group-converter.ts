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
  Affinity,
  AffinityMatchExpression,
  AffinityPriority,
  AffinityType,
  NodeAffinity,
  PodAffinity,
  PodAffinityTerm,
  PreferredNodeSchedulingTerm,
  PreferredPodSchedulingTerm,
  RequiredNodeSchedulingTerm,
  RequiredPodSchedulingTerm,
} from 'shared-types/affinity.types';
import { doesAffinityOperatorRequireValues } from 'utils/db';
import { AffinityCondition, AffinityGroup } from './affinity-group.types';

// Conditions inside a group are AND-ed within a single term, so a group maps to
// one term carrying ALL of its conditions (v1 kept only the first — that loss is
// exactly what #1985 fixes).
const conditionsToMatchExpressions = (
  conditions: AffinityCondition[]
): AffinityMatchExpression[] => {
  const expressions: AffinityMatchExpression[] = [];
  for (const { key, operator, values } of conditions) {
    if (!key || !operator) {
      continue;
    }
    const valuesList = values ? values.filter(Boolean) : [];
    expressions.push({
      key,
      operator,
      ...(doesAffinityOperatorRequireValues(operator) &&
        valuesList.length > 0 && { values: valuesList }),
    });
  }
  return expressions;
};

const matchExpressionsToConditions = (
  matchExpressions: AffinityMatchExpression[] = []
): AffinityCondition[] =>
  matchExpressions.map(({ key, operator, values }) => ({
    key,
    operator,
    values,
  }));

export const groupsToAffinity = (groups: AffinityGroup[]): Affinity => {
  const nodePreferred: PreferredNodeSchedulingTerm[] = [];
  const nodeRequired: RequiredNodeSchedulingTerm = { nodeSelectorTerms: [] };
  const podPreferred: PreferredPodSchedulingTerm[] = [];
  const podRequired: RequiredPodSchedulingTerm = [];
  const antiPreferred: PreferredPodSchedulingTerm[] = [];
  const antiRequired: RequiredPodSchedulingTerm = [];

  for (const group of groups) {
    const matchExpressions = conditionsToMatchExpressions(group.conditions);
    const isRequired = group.priority === AffinityPriority.Required;

    if (group.type === AffinityType.NodeAffinity) {
      if (isRequired) {
        nodeRequired.nodeSelectorTerms.push({ matchExpressions });
      } else {
        nodePreferred.push({
          weight: group.weight ?? 0,
          preference: { matchExpressions },
        });
      }
      continue;
    }

    const term: PodAffinityTerm = {
      topologyKey: group.topologyKey ?? '',
      ...(matchExpressions.length > 0 && {
        labelSelector: { matchExpressions },
      }),
    };
    const preferred =
      group.type === AffinityType.PodAffinity ? podPreferred : antiPreferred;
    const required =
      group.type === AffinityType.PodAffinity ? podRequired : antiRequired;

    if (isRequired) {
      required.push(term);
    } else {
      preferred.push({ weight: group.weight ?? 0, podAffinityTerm: term });
    }
  }

  const nodeAffinity: NodeAffinity = {
    ...(nodePreferred.length > 0 && {
      preferredDuringSchedulingIgnoredDuringExecution: nodePreferred,
    }),
    ...(nodeRequired.nodeSelectorTerms.length > 0 && {
      requiredDuringSchedulingIgnoredDuringExecution: nodeRequired,
    }),
  };
  const podAffinity: PodAffinity = {
    ...(podPreferred.length > 0 && {
      preferredDuringSchedulingIgnoredDuringExecution: podPreferred,
    }),
    ...(podRequired.length > 0 && {
      requiredDuringSchedulingIgnoredDuringExecution: podRequired,
    }),
  };
  const podAntiAffinity: PodAffinity = {
    ...(antiPreferred.length > 0 && {
      preferredDuringSchedulingIgnoredDuringExecution: antiPreferred,
    }),
    ...(antiRequired.length > 0 && {
      requiredDuringSchedulingIgnoredDuringExecution: antiRequired,
    }),
  };

  return {
    ...(Object.keys(nodeAffinity).length > 0 && { nodeAffinity }),
    ...(Object.keys(podAffinity).length > 0 && { podAffinity }),
    ...(Object.keys(podAntiAffinity).length > 0 && { podAntiAffinity }),
  };
};

export const affinityToGroups = (affinity: Affinity): AffinityGroup[] => {
  const groups: AffinityGroup[] = [];
  const { nodeAffinity, podAffinity, podAntiAffinity } = affinity;

  if (nodeAffinity) {
    (
      nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution ?? []
    ).forEach(({ weight, preference }) => {
      groups.push({
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Preferred,
        weight,
        conditions: matchExpressionsToConditions(preference?.matchExpressions),
      });
    });
    (
      nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
        ?.nodeSelectorTerms ?? []
    ).forEach(({ matchExpressions }) => {
      groups.push({
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Required,
        conditions: matchExpressionsToConditions(matchExpressions),
      });
    });
  }

  const podLike: [PodAffinity | undefined, AffinityType][] = [
    [podAffinity, AffinityType.PodAffinity],
    [podAntiAffinity, AffinityType.PodAntiAffinity],
  ];
  for (const [affinityBranch, type] of podLike) {
    if (!affinityBranch) {
      continue;
    }
    (
      affinityBranch.preferredDuringSchedulingIgnoredDuringExecution ?? []
    ).forEach(({ weight, podAffinityTerm }) => {
      groups.push({
        type,
        priority: AffinityPriority.Preferred,
        weight,
        topologyKey: podAffinityTerm.topologyKey,
        conditions: matchExpressionsToConditions(
          podAffinityTerm.labelSelector?.matchExpressions
        ),
      });
    });
    (
      affinityBranch.requiredDuringSchedulingIgnoredDuringExecution ?? []
    ).forEach((term) => {
      groups.push({
        type,
        priority: AffinityPriority.Required,
        topologyKey: term.topologyKey,
        conditions: matchExpressionsToConditions(
          term.labelSelector?.matchExpressions
        ),
      });
    });
  }

  return groups;
};
