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
// own summary in display surfaces (cluster overview, wizard preview) instead of
// being flattened to a scalar row. For a marker widget the value maps each
// target key to the value at its path.
export interface WidgetSummaryProps {
  item: WidgetComponent;
  value: unknown;
}

export interface WidgetSummaryRow {
  label: string;
  text: string;
}

export interface WidgetSummary {
  // Full read-only view (overview card, preview dialog).
  View: ComponentType<WidgetSummaryProps>;
  // Short lines for narrow surfaces (wizard preview); empty = not configured.
  digest: (value: unknown) => WidgetSummaryRow[];
}

export type WidgetSummaryRegistry = Partial<Record<WidgetType, WidgetSummary>>;
