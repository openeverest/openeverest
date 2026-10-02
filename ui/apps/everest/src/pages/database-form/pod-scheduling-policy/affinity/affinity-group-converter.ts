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
  NodeAffinity,
  PodAffinity,
  PodAffinityTerm,
  PreferredNodeSchedulingTerm,
  PreferredPodSchedulingTerm,
  RequiredNodeSchedulingTerm,
  RequiredPodSchedulingTerm,
} from 'shared-types/affinity.types';
import { isPlainObject } from 'components/ui-generator/utils/object-path';
import { AffinityCondition, AffinityGroup } from './affinity-group.types';

type NodeSelectorTerm = RequiredNodeSchedulingTerm['nodeSelectorTerms'][number];

const LABEL_SELECTOR = 'labelSelector';

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

const withPassthrough = (fields: Record<string, unknown>) =>
  Object.keys(fields).length > 0 ? { passthrough: fields } : {};

const nodeTermToGroupFields = ({
  matchExpressions,
  ...rest
}: NodeSelectorTerm) => ({
  conditions: matchExpressionsToConditions(matchExpressions),
  ...withPassthrough(rest),
});

const podTermToGroupFields = ({
  topologyKey,
  labelSelector,
  ...termRest
}: PodAffinityTerm) => {
  const { matchExpressions, ...selectorRest } = labelSelector ?? {
    matchExpressions: [],
  };
  return {
    topologyKey,
    conditions: matchExpressionsToConditions(matchExpressions),
    ...withPassthrough({
      ...termRest,
      ...(Object.keys(selectorRest).length > 0 && {
        [LABEL_SELECTOR]: selectorRest,
      }),
    }),
  };
};

const groupToPodTerm = (
  group: AffinityGroup,
  matchExpressions: AffinityMatchExpression[]
): PodAffinityTerm => {
  const kept: Record<string, unknown> = group.passthrough ?? {};
  const { [LABEL_SELECTOR]: keptSelector, ...keptTerm } = kept;
  const selector = isPlainObject(keptSelector) ? keptSelector : undefined;
  return {
    ...keptTerm,
    topologyKey: group.topologyKey ?? '',
    ...((matchExpressions.length > 0 || selector) && {
      labelSelector: { ...selector, matchExpressions },
    }),
  };
};

const isNodeType = (type: AffinityType) => type === AffinityType.NodeAffinity;

// Kept fields belong to the original term's shape, so they are dropped when an
// edit switches the group between node and pod affinity.
export const applyGroupEdit = (
  original: AffinityGroup,
  edited: AffinityGroup
): AffinityGroup =>
  isNodeType(original.type) === isNodeType(edited.type)
    ? edited
    : { ...edited, passthrough: undefined };

// Kubernetes treats a pod term without a label selector as matching no pods.
export const matchesNoPods = (group: AffinityGroup): boolean =>
  !isNodeType(group.type) &&
  group.conditions.length === 0 &&
  !isPlainObject(group.passthrough?.[LABEL_SELECTOR]);

export const getPassthroughFieldNames = (
  passthrough: Record<string, unknown> = {}
): string[] =>
  Object.entries(passthrough).flatMap(([key, value]) =>
    key === LABEL_SELECTOR && isPlainObject(value) ? Object.keys(value) : [key]
  );

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
      const term = { ...group.passthrough, matchExpressions };
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

    const term = groupToPodTerm(group, matchExpressions);
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
        ...nodeTermToGroupFields(preference ?? { matchExpressions: [] }),
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
