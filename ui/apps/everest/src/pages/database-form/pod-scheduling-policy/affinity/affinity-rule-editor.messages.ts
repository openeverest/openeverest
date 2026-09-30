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

import { AffinityPriority } from 'shared-types/affinity.types';

export const Messages = {
  label: 'Affinity',
  groupsInfo:
    "Conditions in a rule group are combined with AND. Required Node affinity groups are alternatives: a node must match at least one of them (OR). All other Required groups must hold together. Preferred groups never block scheduling: each one a node satisfies adds its weight to that node's score.",
  priorityHeading: {
    [AffinityPriority.Required]:
      'Required — pods are scheduled only where these hold',
    [AffinityPriority.Preferred]:
      'Preferred — nodes that satisfy these score higher; never blocks scheduling',
  } satisfies Record<AffinityPriority, string>,
  weight: (weight: number) => `+${weight}`,
  addGroup: 'Add rule group',
  empty: 'No affinity rules yet. Add a rule group to get started.',
  emptyReadOnly: 'No affinity rules for this component.',
  or: 'OR',
  and: 'and',
};
