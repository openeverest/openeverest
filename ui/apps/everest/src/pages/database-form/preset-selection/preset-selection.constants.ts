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

import { CardPickerBadge } from '@percona/ui-lib';
import { PresetScope } from './preset-selection.types';

// Records which preset an Instance was created from (provenance). The server
// accepts only one of the two on an Instance and checks the spec against the
// preset of that kind.
export const PRESET_ANNOTATION_BY_SCOPE: Record<PresetScope, string> = {
  [PresetScope.Cluster]: 'openeverest.io/cluster-instance-preset',
  [PresetScope.Namespace]: 'openeverest.io/instance-preset',
};

export const PRESET_BADGE_COLOR_BY_SCOPE: Record<
  PresetScope,
  CardPickerBadge['color']
> = {
  [PresetScope.Cluster]: 'default',
  [PresetScope.Namespace]: 'info',
};
