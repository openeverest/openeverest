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

import { MenuItem, Typography } from '@mui/material';
import { SelectInput } from '@percona/ui-lib';
import { DbWizardFormFields } from 'consts';
import { usePresetSelectionContext } from './preset-selection.context';
import { Messages } from './preset-select.messages';

export const PresetSelect = () => {
  const {
    presets,
    presetName,
    isLoadingPresets,
    isResolving,
    isError,
    resolveError,
  } = usePresetSelectionContext();

  // Hide the picker only when the provider genuinely has no presets — keep it
  // visible on load errors so the user knows the feature exists but failed.
  if (!isLoadingPresets && !isError && presets.length === 0) {
    return null;
  }

  const helperText = resolveError
    ? resolveError
    : isError
      ? Messages.loadError
      : presetName
        ? Messages.selectedCaption
        : Messages.helper;

  return (
    <SelectInput
      name={DbWizardFormFields.presetName}
      label={Messages.label}
      loading={isLoadingPresets || isResolving}
      helperText={helperText}
      error={isError || Boolean(resolveError)}
      formControlProps={{ sx: { mt: 3 } }}
      selectFieldProps={{
        displayEmpty: true,
        disabled: isError && presets.length === 0,
        renderValue: (value: unknown) =>
          value ? String(value) : Messages.none,
      }}
    >
      <MenuItem value="">{Messages.none}</MenuItem>
      {presets.map((preset) => {
        const name = preset.metadata?.name ?? '';
        const topology = preset.spec?.topology?.type;
        const version = preset.spec?.version;
        const details = [topology, version].filter(Boolean).join(' · ');
        return (
          <MenuItem key={name} value={name}>
            {name}
            {details && (
              <Typography
                component="span"
                variant="caption"
                color="text.secondary"
                sx={{ ml: 1 }}
              >
                {details}
              </Typography>
            )}
          </MenuItem>
        );
      })}
    </SelectInput>
  );
};
