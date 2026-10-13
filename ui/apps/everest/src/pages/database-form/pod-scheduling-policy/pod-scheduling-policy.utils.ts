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

import { Affinity, AffinityPriority } from 'shared-types/affinity.types';
import { isPlainObject } from 'components/ui-generator/utils/object-path';
import { WidgetSummaryRow } from 'components/ui-generator/widget-summary-registry';
import { AffinityGroup } from './affinity/affinity-group.types';
import { affinityToGroups } from './affinity/affinity-group-converter';
import { Messages } from './pod-scheduling-policy-section.messages';

export interface ComponentAffinityGroups {
  key: string;
  groups: AffinityGroup[];
}

const isAffinity = (value: unknown): value is Affinity =>
  typeof value === 'object' && value !== null;

export const toAffinityGroups = (value: unknown): AffinityGroup[] =>
  affinityToGroups(isAffinity(value) ? value : {});

// The marker's value maps each target component to its saved Affinity.
export const getComponentAffinityGroups = (
  value: unknown
): ComponentAffinityGroups[] =>
  Object.entries(isPlainObject(value) ? value : {}).map(([key, affinity]) => ({
    key,
    groups: toAffinityGroups(affinity),
  }));

export const describeGroupCounts = (groups: AffinityGroup[]): string => {
  const required = groups.filter(
    ({ priority }) => priority === AffinityPriority.Required
  ).length;
  const preferred = groups.length - required;
  return [
    required > 0 ? Messages.required(required) : '',
    preferred > 0 ? Messages.preferred(preferred) : '',
  ]
    .filter(Boolean)
    .join(', ');
};

// Only components that carry rules; an empty result means nothing is configured.
export const digestSchedulingPolicy = (value: unknown): WidgetSummaryRow[] =>
  getComponentAffinityGroups(value)
    .filter(({ groups }) => groups.length > 0)
    .map(({ key, groups }) => ({
      label: key,
      text: describeGroupCounts(groups),
    }));
