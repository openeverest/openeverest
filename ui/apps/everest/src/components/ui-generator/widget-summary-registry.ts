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

import type { ComponentType } from 'react';
import type { WidgetComponent, WidgetType } from './ui-generator.types';

// Read-only counterpart of WidgetRegistry: a widget-typed component renders its
// own summary in display surfaces (e.g. the cluster overview) instead of being
// flattened to a scalar row. Value is the raw field value read from the instance.
export interface WidgetSummaryProps {
  item: WidgetComponent;
  value: unknown;
}

export type WidgetSummary = ComponentType<WidgetSummaryProps>;

export type WidgetSummaryRegistry = Partial<Record<WidgetType, WidgetSummary>>;
