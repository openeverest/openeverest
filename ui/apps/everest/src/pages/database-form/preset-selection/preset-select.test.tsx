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

import { render, screen } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { PresetSelect } from './preset-select';
import { PresetSelectionContext } from './preset-selection.context';
import { PresetSelectionContextType } from './preset-selection.types';

const baseContext: PresetSelectionContextType = {
  presets: [],
  isLoadingPresets: false,
  isError: false,
  presetName: '',
  resolvedPreset: null,
  isResolving: false,
  resolveError: null,
  presetSelected: false,
  presetPending: false,
};

const PICKER_TESTID = 'select-input-preset-name';

const renderPicker = (overrides: Partial<PresetSelectionContextType>) => {
  const value = { ...baseContext, ...overrides };
  const Harness = () => {
    const methods = useForm({ defaultValues: { presetName: '' } });
    return (
      <TestWrapper>
        <PresetSelectionContext.Provider value={value}>
          <FormProvider {...methods}>
            <PresetSelect />
          </FormProvider>
        </PresetSelectionContext.Provider>
      </TestWrapper>
    );
  };
  return render(<Harness />);
};

describe('PresetSelect', () => {
  it('is hidden when the provider has no presets (A2)', () => {
    renderPicker({ presets: [] });

    expect(screen.queryByTestId(PICKER_TESTID)).not.toBeInTheDocument();
  });

  it('stays visible on a load error so the user knows the feature exists (A3)', () => {
    renderPicker({ presets: [], isError: true });

    expect(screen.getByTestId(PICKER_TESTID)).toBeInTheDocument();
  });
});
