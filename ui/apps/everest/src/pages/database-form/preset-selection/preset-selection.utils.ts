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
import { INSTANCE_PRESET_ANNOTATION } from './preset-selection.constants';

interface BuildPresetCreateArgsInput {
  resolvedPreset: InstancePreset;
  provider: string;
  dbName: string;
  k8sNamespace: string;
  presetName: string;
}

interface PresetCreateArgs {
  formValue: {
    provider: string;
    dbName: string;
    k8sNamespace: string;
    spec: InstancePreset['spec'];
  };
  annotations: Record<string, string>;
}

// Builds the create-instance arguments for a one-click preset deploy: the
// resolved preset spec is sent verbatim (never rebuilt from the wizard fields)
// and the originating preset is recorded in an annotation. Only request-address
// fields ride alongside the spec; they are stripped before submit by
// buildCreateInstanceSpec, so no wizard-only value leaks into the Instance spec.
export const buildPresetCreateArgs = ({
  resolvedPreset,
  provider,
  dbName,
  k8sNamespace,
  presetName,
}: BuildPresetCreateArgsInput): PresetCreateArgs => ({
  formValue: {
    provider,
    dbName,
    k8sNamespace,
    spec: resolvedPreset.spec,
  },
  annotations: { [INSTANCE_PRESET_ANNOTATION]: presetName },
});
