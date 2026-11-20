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
import { PresetSelectCards } from './preset-select-cards';
import {
  PresetSelectionContext,
  PresetSelectionContextType,
} from '../preset-selection-context';
import { PresetScope, SelectablePreset } from '../preset-selection.types';
import { getClusterPresetId } from '../preset-selection.utils';

const baseContext: PresetSelectionContextType = {
  presets: [],
  isLoadingPresets: false,
  isError: false,
  presetId: '',
  selectedPreset: null,
  presetSelected: false,
  presetPending: false,
  presetUnavailable: false,
  droppedPreset: null,
};

const preset: SelectablePreset = {
  id: getClusterPresetId('psmdb-x'),
  scope: PresetScope.Cluster,
  name: 'psmdb-x',
  spec: {
    providerRef: { name: 'percona-server-mongodb' },
    version: '8.0.12',
    topology: { type: 'replicaSet' },
    components: {
      engine: {
        replicas: 3,
        resources: { limits: { cpu: 1, memory: '4Gi' } },
        storage: { size: '25Gi' },
      },
    },
  },
};

const renderCards = (overrides: Partial<PresetSelectionContextType>) => {
  const value = { ...baseContext, ...overrides };
  const Harness = () => {
    const methods = useForm({ defaultValues: { presetId: '' } });
    return (
      <TestWrapper>
        <PresetSelectionContext.Provider value={value}>
          <FormProvider {...methods}>
            <PresetSelectCards />
          </FormProvider>
        </PresetSelectionContext.Provider>
      </TestWrapper>
    );
  };
  return render(<Harness />);
};

describe('PresetSelectCards', () => {
  it('renders no section when the provider has no presets (A2)', () => {
    const { container } = renderCards({ presets: [] });

    expect(container).toBeEmptyDOMElement();
  });

  it('renders no section while presets are still loading', () => {
    const { container } = renderCards({ isLoadingPresets: true, presets: [] });

    expect(container).toBeEmptyDOMElement();
  });

  it('renders the scratch card and wires presets into the picker', () => {
    renderCards({ presets: [preset] });

    expect(
      screen.getByRole('button', { name: /Start from scratch/ })
    ).toBeInTheDocument();
    // jsdom width is 0 -> a single column -> the preset overflows to Browse,
    // which proves the preset was passed through to the picker.
    expect(
      screen.getByRole('button', { name: /Browse all 1 presets/ })
    ).toBeInTheDocument();
  });

  it('shows a disabled notice card when the presets fail to load (A3)', () => {
    renderCards({ presets: [], isError: true });

    // Failure is surfaced in-grid as a disabled card beside "start from
    // scratch", not as an out-of-flow error line.
    expect(screen.getByText(/Presets didn't load/i)).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /Start from scratch/ })
    ).toBeInTheDocument();
  });

  it('keeps the loaded presets and says only some failed when one list fails', () => {
    renderCards({ presets: [preset], isError: true });

    expect(screen.getByText(/Some presets didn't load/i)).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /Browse all 1 presets/ })
    ).toBeInTheDocument();
  });

  it('explains why a namespace preset was dropped after a namespace switch', () => {
    renderCards({
      presets: [preset],
      droppedPreset: {
        name: 'psmdb-x',
        namespace: 'team-b',
        presetNamespace: 'team-a',
      },
    });

    // Names the owning namespace: "team-b" may have its own "psmdb-x".
    expect(
      screen.getByText(
        "Preset “psmdb-x” belongs to namespace “team-a” and can't be used in “team-b”."
      )
    ).toBeInTheDocument();
  });

  it('still explains the drop when the new namespace offers no presets at all', () => {
    renderCards({
      presets: [],
      droppedPreset: {
        name: 'psmdb-x',
        namespace: 'team-b',
        presetNamespace: 'team-a',
      },
    });

    expect(
      screen.getByText(/belongs to namespace “team-a”/)
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /Start from scratch/ })
    ).toBeInTheDocument();
  });

  it('says a preset is gone when it disappeared without a namespace switch', () => {
    renderCards({
      presets: [preset],
      droppedPreset: {
        name: 'psmdb-old',
        namespace: 'team-a',
        presetNamespace: 'team-a',
      },
    });

    expect(
      screen.getByText('Preset “psmdb-old” is no longer available.')
    ).toBeInTheDocument();
  });
});
