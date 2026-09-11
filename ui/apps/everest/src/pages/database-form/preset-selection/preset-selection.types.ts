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

import { InstancePreset } from 'shared-types/api.types';

export interface PresetSelectionContextType {
  presets: InstancePreset[];
  isLoadingPresets: boolean;
  isError: boolean;
  presetName: string;
  resolvedPreset: InstancePreset | null;
  isResolving: boolean;
  resolveError: string | null;
  presetSelected: boolean;
  // True while a preset is picked but its spec is not yet resolved (or failed):
  // the one-click create must stay blocked until we have a spec to submit.
  presetPending: boolean;
}
