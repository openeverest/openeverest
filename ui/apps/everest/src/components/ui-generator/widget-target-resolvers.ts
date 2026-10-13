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

import { Provider } from 'shared-types/api.types';
import { resolveAffinityTargets } from 'utils/pod-scheduling-policy';
import { WidgetTarget, WidgetType } from './ui-generator.types';

export interface WidgetTargetContext {
  providerObject?: Provider;
  topology: string;
}

// Marker widgets whose written API paths depend on the provider; resolved once
// in preprocess so every stage (toggleable, payload, overview) sees them.
export const widgetTargetResolvers: Partial<
  Record<WidgetType, (context: WidgetTargetContext) => WidgetTarget[]>
> = {
  [WidgetType.PodSchedulingPolicy]: ({ providerObject, topology }) =>
    resolveAffinityTargets(providerObject, topology),
};
