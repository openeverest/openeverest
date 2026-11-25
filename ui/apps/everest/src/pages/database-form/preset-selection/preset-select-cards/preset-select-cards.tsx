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

import { useMemo } from 'react';
import { Alert, Box } from '@mui/material';
import { useFormContext } from 'react-hook-form';
import { CardPicker } from '@percona/ui-lib';
import { DbWizardFormFields } from 'consts';
import { usePresetSelectionContext } from '../preset-selection-context';
import {
  getDroppedPresetMessage,
  presetToCardItem,
} from './preset-select-cards.utils';
import { Messages } from './preset-select-cards.messages';

// Thin preset-specific wrapper over the reusable CardPicker: maps presets to
// card items (searchable by their whole spec) and binds selection to the form.
export const PresetSelectCards = () => {
  const {
    presets,
    presetId,
    isLoadingPresets,
    isError,
    droppedPreset,
    dismissDroppedPreset,
  } = usePresetSelectionContext();
  const { setValue } = useFormContext();

  const items = useMemo(() => presets.map(presetToCardItem), [presets]);

  // A dropped pick keeps the section up even with nothing left to offer, or
  // the form would fall back to defaults with no explanation.
  if (
    !isError &&
    !droppedPreset &&
    (isLoadingPresets || presets.length === 0)
  ) {
    return null;
  }

  const statusCard = isError
    ? {
        title:
          presets.length > 0
            ? Messages.partialLoadErrorTitle
            : Messages.loadErrorTitle,
        subtitle: Messages.loadErrorSubtitle,
      }
    : undefined;

  return (
    <Box sx={{ mt: 3 }}>
      <CardPicker
        items={items}
        selectedId={presetId}
        onSelect={(id) =>
          setValue(DbWizardFormFields.presetId, id, {
            shouldValidate: true,
            shouldDirty: true,
            shouldTouch: true,
          })
        }
        leadCard={{
          id: '',
          title: Messages.scratchTitle,
          subtitle: Messages.scratchCaption,
        }}
        statusCard={statusCard}
        messages={{
          searchPlaceholder: Messages.searchPlaceholder,
          searchAriaLabel: Messages.searchAriaLabel,
          browseAll: Messages.browseAll,
          dialogTitle: Messages.dialogTitle,
          noMatches: Messages.noMatches,
        }}
      />
      {/* Not an error: the form is valid, the pick just no longer applies. */}
      {droppedPreset && (
        <Alert
          severity="info"
          role="status"
          onClose={dismissDroppedPreset}
          sx={{ mt: 2 }}
        >
          {getDroppedPresetMessage(droppedPreset)} {Messages.presetDroppedHint}
        </Alert>
      )}
    </Box>
  );
};
