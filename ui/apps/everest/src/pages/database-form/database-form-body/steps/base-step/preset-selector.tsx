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

import { AutoCompleteInput } from '@percona/ui-lib';
import { DbWizardFormFields } from 'consts';
import { useInstancePresets } from 'hooks/api/instance-presets/useInstancePresets';
import { useMemo } from 'react';
import { InstancePreset } from 'shared-types/api.types';

export const Messages = {
  labels: {
    preset: 'Template',
  },
  noPresets: 'No templates available for this provider',
};

interface PresetSelectorProps {
  provider?: string;
  disabled?: boolean;
}

export const PresetSelector = ({ provider, disabled }: PresetSelectorProps) => {
  const { data: presetsData, isLoading } = useInstancePresets(provider);

  const presetOptions = useMemo(() => {
    if (!presetsData?.items) return [];
    return presetsData.items.map((preset: InstancePreset) => ({
      label: preset.metadata?.name || '',
      value: preset.metadata?.name || '',
    }));
  }, [presetsData]);

  return (
    <AutoCompleteInput
      name={DbWizardFormFields.instancePreset}
      label={Messages.labels.preset}
      loading={isLoading}
      disabled={disabled || isLoading}
      options={presetOptions}
      autoCompleteProps={{
        isOptionEqualToValue: (
          option: { label: string; value: string },
          value: { label: string; value: string } | string
        ) => {
          return (
            option.value === (typeof value === 'string' ? value : value?.value)
          );
        },
        noOptionsText: Messages.noPresets,
        disableClearable: false,
      }}
    />
  );
};
