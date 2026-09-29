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
import { AffinityRuleEditor } from 'pages/database-form/pod-scheduling-policy/affinity';
import { AffinitySummary } from '../affinity-summary';

// Widgets the overview can edit (in SectionEditModal) and display read-only
// (in schema-driven cards). Mirrors the wizard's per-section registries.
export const overviewWidgetRegistry: WidgetRegistry = {
  [WidgetType.Affinity]: AffinityRuleEditor,
};

export const overviewWidgetSummaryRegistry: WidgetSummaryRegistry = {
  [WidgetType.Affinity]: AffinitySummary,
};
