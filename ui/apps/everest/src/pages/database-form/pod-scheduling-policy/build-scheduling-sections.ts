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
  Section,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';
import {
  SchedulingPolicyType,
  SchedulingSupportByComponent,
} from 'shared-types/podSchedulingPolicy.types';

const AFFINITY_POLICY: SchedulingPolicyType = 'affinity';

export const schedulingPolicyPath = (
  component: string,
  policy: SchedulingPolicyType
): string => `spec.components.${component}.schedulingPolicy.${policy}`;

// One section per component that supports affinity; each holds a single affinity
// widget bound to that component's schedulingPolicy.affinity path, so the widget
// reads and writes the payload directly through the engine's name resolution.
export const buildAffinitySections = (
  support: SchedulingSupportByComponent
): { [component: string]: Section } => {
  const sections: { [component: string]: Section } = {};
  for (const [component, policies] of Object.entries(support)) {
    if (!policies.includes(AFFINITY_POLICY)) {
      continue;
    }
    sections[component] = {
      components: {
        [AFFINITY_POLICY]: {
          uiType: WidgetType.Affinity,
          path: schedulingPolicyPath(component, AFFINITY_POLICY),
          fieldParams: {},
        },
      },
    };
  }
  return sections;
};
