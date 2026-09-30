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
  WidgetRegistry,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';
import { WidgetSummaryRegistry } from 'components/ui-generator/widget-summary-registry';
import { PodSchedulingPolicySection } from './pod-scheduling-policy';
import { SchedulingPolicyView } from './pod-scheduling-policy/scheduling-policy-view';
import { digestSchedulingPolicy } from './pod-scheduling-policy/pod-scheduling-policy.utils';

// Binds the wizard's schema-driven widget markers to their renderers. One place
// to register future wizard widgets (tolerations, node selector, …).
export const wizardWidgetRegistry: WidgetRegistry = {
  [WidgetType.PodSchedulingPolicy]: PodSchedulingPolicySection,
};

// Read-only rendering of the same widgets, shared by the wizard preview and the
// cluster overview so both show identical content.
export const widgetSummaryRegistry: WidgetSummaryRegistry = {
  [WidgetType.PodSchedulingPolicy]: {
    View: SchedulingPolicyView,
    digest: digestSchedulingPolicy,
  },
};
