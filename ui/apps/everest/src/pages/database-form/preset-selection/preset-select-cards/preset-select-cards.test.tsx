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

import { fireEvent, render, screen } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import {
  FieldType,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { DatabaseFormProvider } from 'pages/database-form/database-form-context';
import { TestWrapper } from 'utils/test';
import { PresetSelectCards } from './preset-select-cards';
import {
  PresetSelectionContext,
  PresetSelectionContextType,
} from '../preset-selection-context';
import { TEAM_A, TEAM_B, clusterPreset } from '../__mocks__/presets';
import { noPresetsContext } from '../__mocks__/preset-selection-context';

const renderCards = (
  overrides: Partial<PresetSelectionContextType>,
  uiSchema: TopologyUISchemas = {}
) => {
  const value = { ...noPresetsContext, ...overrides };
  const topologies = Object.keys(uiSchema);
  const Harness = () => {
    const methods = useForm({ defaultValues: { presetId: '' } });
    return (
      <TestWrapper>
        <DatabaseFormProvider
          value={{
            uiSchema,
            topologies,
            defaultTopology: topologies[0] ?? '',
            hasMultipleTopologies: topologies.length > 1,
            sections: {},
            hasBackupStep: false,
            hasMonitoringStep: false,
          }}
        >
          <PresetSelectionContext.Provider value={value}>
            <FormProvider {...methods}>
              <PresetSelectCards />
            </FormProvider>
          </PresetSelectionContext.Provider>
        </DatabaseFormProvider>
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
    renderCards({ presets: [clusterPreset] });

    expect(
      screen.getByRole('button', { name: /Start from scratch/ })
    ).toBeInTheDocument();
    // jsdom width is 0 -> a single column -> the preset overflows to Browse,
    // which proves the preset was passed through to the picker.
    expect(
      screen.getByRole('button', { name: /Browse all 1 presets/ })
    ).toBeInTheDocument();
  });

  it('groups resource values by component using the preset topology labels and units', () => {
    renderCards(
      {
        presets: [
          {
            ...clusterPreset,
            spec: {
              ...clusterPreset.spec,
              providerRef: { name: 'test-provider' },
              topology: { type: 'sharded' },
              components: {
                primary: {
                  replicas: 3,
                  resources: { limits: { cpu: 1, memory: '2048Mi' } },
                  storage: { size: '644245094400m' },
                },
                router: { replicas: 2, version: '1.0.0' },
                configServer: { replicas: 3 },
              },
            },
          },
        ],
      },
      {
        replicaSet: { sections: {} },
        sharded: {
          sectionsOrder: ['sizing'],
          sections: {
            storage: {
              components: {
                disk: {
                  uiType: FieldType.Number,
                  path: 'spec.components.primary.storage.size',
                  fieldParams: {
                    label: 'Disk',
                    badge: 'Gi',
                    badgeToApi: true,
                  },
                },
              },
            },
            sizing: {
              components: {
                primary: {
                  uiType: 'group',
                  componentsOrder: ['nodes', 'memory'],
                  components: {
                    memory: {
                      uiType: FieldType.Quantity,
                      path: 'spec.components.primary.resources.limits.memory',
                      fieldParams: {
                        label: 'Memory',
                        unit: 'Gi',
                        unitLabel: 'GiB',
                      },
                    },
                    nodes: {
                      uiType: FieldType.Number,
                      path: [
                        'spec.components.primary.replicas',
                        'spec.components.router.replicas',
                      ],
                      fieldParams: { label: 'Nodes' },
                    },
                  },
                },
                routerVersion: {
                  uiType: FieldType.Text,
                  path: 'spec.components.router.version',
                  fieldParams: { label: 'Router version' },
                },
              },
            },
          },
        },
      }
    );

    fireEvent.click(
      screen.getByRole('button', { name: /Browse all 1 presets/ })
    );

    expect(
      screen.getByText(
        'primary (Nodes: 3 · Memory: 2 GiB · Disk: 0.6 Gi · 1 CPU) | router (Nodes: 2) | configServer (3 nodes)'
      )
    ).toBeInTheDocument();
  });

  it('reports a failed load under the cards and offers to retry (A3)', () => {
    const retryFailedLists = vi.fn();
    renderCards({ presets: [], isError: true, retryFailedLists });

    // "Presets", not "Some presets": nothing loaded at all.
    expect(screen.getByRole('alert')).toHaveTextContent(
      /^Presets didn't load\./
    );
    expect(
      screen.getByRole('button', { name: /Start from scratch/ })
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));

    expect(retryFailedLists).toHaveBeenCalledTimes(1);
  });

  it('keeps the loaded presets and says only some failed when one list fails', () => {
    renderCards({ presets: [clusterPreset], isError: true });

    expect(screen.getByRole('alert')).toHaveTextContent(
      /^Some presets didn't load\./
    );
    expect(
      screen.getByRole('button', { name: /Browse all 1 presets/ })
    ).toBeInTheDocument();
  });

  it('holds the retry button while the failed list is requested again', () => {
    renderCards({ presets: [clusterPreset], isError: true, isRetrying: true });

    expect(screen.getByRole('button', { name: /Retry/ })).toBeDisabled();
  });

  it('still explains the drop when the new namespace offers no presets at all', () => {
    renderCards({
      presets: [],
      droppedPreset: {
        name: clusterPreset.name,
        wizardNamespace: TEAM_B,
        ownerNamespace: TEAM_A,
      },
    });

    expect(
      screen.getByText(/belongs to namespace “team-a”/)
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /Start from scratch/ })
    ).toBeInTheDocument();
  });

  it('lets the user dismiss the dropped-preset notice', () => {
    const dismissDroppedPreset = vi.fn();
    renderCards({
      presets: [clusterPreset],
      droppedPreset: {
        name: clusterPreset.name,
        wizardNamespace: TEAM_B,
        ownerNamespace: TEAM_A,
      },
      dismissDroppedPreset,
    });

    fireEvent.click(screen.getByRole('button', { name: 'Close' }));

    expect(dismissDroppedPreset).toHaveBeenCalledTimes(1);
  });
});
