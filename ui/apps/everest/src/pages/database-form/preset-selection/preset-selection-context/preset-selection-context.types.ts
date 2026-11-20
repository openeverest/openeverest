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

import { DroppedPreset, SelectablePreset } from '../preset-selection.types';

export interface PresetSelectionContextType {
  presets: SelectablePreset[];
  isLoadingPresets: boolean;
  // True when either list failed; the other one is still offered.
  isError: boolean;
  presetId: string;
  selectedPreset: SelectablePreset | null;
  presetSelected: boolean;
  // True while a preset is picked but its spec is not yet available:
  // the one-click create must stay blocked until we have a spec to submit.
  presetPending: boolean;
  // True when the picked preset cannot be resolved any more (the namespace it
  // lives in was switched away from, or it was removed): the pick must be
  // dropped, or the create would stay blocked with no explanation.
  presetUnavailable: boolean;
  droppedPreset: DroppedPreset | null;
}
