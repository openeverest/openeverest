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
  AffinityOperator,
  AffinityPriority,
  AffinityType,
  MatchExpressionsSelector,
  NodeAffinity,
  NodeSelectorTerm,
  PodAffinity,
  PodAffinityTerm,
  PreferredNodeSchedulingTerm,
  PreferredPodSchedulingTerm,
  RequiredNodeSchedulingTerm,
  RequiredPodSchedulingTerm,
} from 'shared-types/affinity.types';
import { isPlainObject } from 'components/ui-generator/utils/object-path';
import { AffinityCondition, AffinityGroup } from './affinity-group.types';

const MATCH_EXPRESSIONS = 'matchExpressions';
const TOPOLOGY_KEY = 'topologyKey';

// Blocklist rather than allowlist: operators set outside the UI (node Gt / Lt)
// must keep their values too.
const VALUELESS_OPERATORS: string[] = [
  AffinityOperator.Exists,
  AffinityOperator.DoesNotExist,
];

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
      ...(!VALUELESS_OPERATORS.includes(operator) &&
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

const nodeTermToGroupFields = (term: NodeSelectorTerm) => ({
  conditions: matchExpressionsToConditions(term.matchExpressions),
  source: term,
});

const podTermToGroupFields = (term: PodAffinityTerm) => ({
  topologyKey: term.topologyKey,
  conditions: matchExpressionsToConditions(
    term.labelSelector?.matchExpressions
  ),
  source: term,
});

const isNodeType = (type: AffinityType) => type === AffinityType.NodeAffinity;

const isEmptyValue = (value: unknown): boolean =>
  Array.isArray(value)
    ? value.length === 0
    : isPlainObject(value) && Object.keys(value).length === 0;

const isPodTerm = (
  term: NodeSelectorTerm | PodAffinityTerm
): term is PodAffinityTerm => TOPOLOGY_KEY in term;

// A source of the other kind is ignored: an edit may switch node <-> pod.
const nodeSourceOf = ({
  type,
  source,
}: AffinityGroup): NodeSelectorTerm | undefined =>
  isNodeType(type) && source && !isPodTerm(source) ? source : undefined;

const podSourceOf = ({
  type,
  source,
}: AffinityGroup): PodAffinityTerm | undefined =>
  !isNodeType(type) && source && isPodTerm(source) ? source : undefined;

// Writes the editor's expressions into the selector as read, keeping the rest.
// Absent and empty selectors differ in k8s (no pods vs. all pods), so with no
// expressions to write the selector stays exactly as read.
const withMatchExpressions = (
  selector: MatchExpressionsSelector | undefined,
  matchExpressions: AffinityMatchExpression[]
): MatchExpressionsSelector | undefined => {
  const {
    matchExpressions: readExpressions,
    ...rest
  }: MatchExpressionsSelector = selector ?? {};
  if (matchExpressions.length > 0) {
    return { ...rest, matchExpressions };
  }
  if (!readExpressions?.length) {
    return selector;
  }
  // Every condition was removed: only the selector's other fields remain.
  return Object.keys(rest).length > 0 ? rest : undefined;
};

const groupToNodeTerm = (group: AffinityGroup): NodeSelectorTerm =>
  withMatchExpressions(
    nodeSourceOf(group),
    conditionsToMatchExpressions(group.conditions)
  ) ?? {};

const groupToPodTerm = (group: AffinityGroup): PodAffinityTerm => {
  const { labelSelector: readSelector, ...readTerm }: Partial<PodAffinityTerm> =
    podSourceOf(group) ?? {};
  const labelSelector = withMatchExpressions(
    readSelector,
    conditionsToMatchExpressions(group.conditions)
  );
  return {
    ...readTerm,
    topologyKey: group.topologyKey ?? '',
    ...(labelSelector && { labelSelector }),
  };
};

// Kubernetes treats a pod term without a label selector as matching no pods.
export const matchesNoPods = (group: AffinityGroup): boolean =>
  !isNodeType(group.type) && !groupToPodTerm(group).labelSelector;

// ...and a selector with no requirements as matching every pod.
export const matchesAllPods = (group: AffinityGroup): boolean => {
  const selector = isNodeType(group.type)
    ? undefined
    : groupToPodTerm(group).labelSelector;
  return !!selector && Object.values(selector).every(isEmptyValue);
};

// Without conditions a group is kept only if its term carries something the
// editor doesn't model (matchFields, matchLabels, namespaces, an empty selector).
export const canSaveWithoutConditions = (group: AffinityGroup): boolean => {
  const withoutConditions = { ...group, conditions: [] };
  const ownedKey = isNodeType(group.type) ? MATCH_EXPRESSIONS : TOPOLOGY_KEY;
  const term = isNodeType(group.type)
    ? groupToNodeTerm(withoutConditions)
    : groupToPodTerm(withoutConditions);
  return Object.keys(term).some((key) => key !== ownedKey);
};

export const groupsToAffinity = (groups: AffinityGroup[]): Affinity => {
  const nodePreferred: PreferredNodeSchedulingTerm[] = [];
  const nodeRequired: RequiredNodeSchedulingTerm = { nodeSelectorTerms: [] };
  const podPreferred: PreferredPodSchedulingTerm[] = [];
  const podRequired: RequiredPodSchedulingTerm = [];
  const antiPreferred: PreferredPodSchedulingTerm[] = [];
  const antiRequired: RequiredPodSchedulingTerm = [];

  for (const group of groups) {
    const isRequired = group.priority === AffinityPriority.Required;

    if (group.type === AffinityType.NodeAffinity) {
      const term = groupToNodeTerm(group);
      if (isRequired) {
        nodeRequired.nodeSelectorTerms.push(term);
      } else {
        nodePreferred.push({
          weight: group.weight ?? 0,
          preference: term,
        });
      }
      continue;
    }

    const term = groupToPodTerm(group);
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
        ...nodeTermToGroupFields(preference ?? {}),
      });
    });
    (
      nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
        ?.nodeSelectorTerms ?? []
    ).forEach((term) => {
      groups.push({
        type: AffinityType.NodeAffinity,
        priority: AffinityPriority.Required,
        ...nodeTermToGroupFields(term),
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
        ...podTermToGroupFields(podAffinityTerm),
      });
    });
    (
      affinityBranch.requiredDuringSchedulingIgnoredDuringExecution ?? []
    ).forEach((term) => {
      groups.push({
        type,
        priority: AffinityPriority.Required,
        ...podTermToGroupFields(term),
      });
    });
  }

  return groups;
};
