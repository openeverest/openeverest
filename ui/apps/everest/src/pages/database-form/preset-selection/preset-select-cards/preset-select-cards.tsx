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
import { useFormContext } from 'react-hook-form';
import { CardPicker } from '@percona/ui-lib';
import { DbWizardFormFields } from 'consts';
import { usePresetSelectionContext } from '../preset-selection-context';
import { presetToCardItem } from './preset-select-cards.utils';
import { Messages } from './preset-select-cards.messages';

// Thin preset-specific wrapper over the reusable CardPicker: maps presets to
// card items (searchable by their whole spec) and binds selection to the form.
export const PresetSelectCards = () => {
  const { presets, presetName, isLoadingPresets, isError, resolveError } =
    usePresetSelectionContext();
  const { setValue } = useFormContext();

  const items = useMemo(() => presets.map(presetToCardItem), [presets]);

  if (!isError && (isLoadingPresets || presets.length === 0)) {
    return null;
  }

  const statusCard = isError
    ? { title: Messages.loadErrorTitle, subtitle: Messages.loadErrorSubtitle }
    : undefined;

  return (
    <CardPicker
      sx={{ mb: 2 }}
      items={items}
      selectedId={presetName}
      onSelect={(id) =>
        setValue(DbWizardFormFields.presetName, id, {
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
      error={resolveError ?? undefined}
      messages={{
        searchPlaceholder: Messages.searchPlaceholder,
        searchAriaLabel: Messages.searchAriaLabel,
        browseAll: Messages.browseAll,
        dialogTitle: Messages.dialogTitle,
        noMatches: Messages.noMatches,
      }}
    />
  );
};
