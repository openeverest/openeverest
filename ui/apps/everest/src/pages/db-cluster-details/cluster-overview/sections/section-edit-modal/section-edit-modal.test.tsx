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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { TestWrapper } from 'utils/test';
import type {
  Component,
  Section,
} from 'components/ui-generator/ui-generator.types';
import {
  FieldType,
  FormMode,
  GroupType,
} from 'components/ui-generator/ui-generator.types';
import type { Instance, Provider } from 'shared-types/api.types';
import { preprocessSchema } from 'components/ui-generator/utils/preprocess/preprocess-schema';
import { Messages } from 'components/ui-generator/api-providers/data-source-field/data-source-field.messages';
import { Messages as ProvidersMessages } from 'components/ui-generator/api-providers/providers.messages';
import SectionEditModal from './section-edit-modal';

const { mutate, clusterInfo } = vi.hoisted(() => {
  const clusterInfoResult: { data?: { storageClassNames: string[] } } = {};
  return { mutate: vi.fn(), clusterInfo: clusterInfoResult };
});

vi.mock('hooks/api/kubernetesClusters/useKubernetesClusterInfo', () => ({
  useKubernetesClusterInfo: () => clusterInfo,
}));

vi.mock('hooks/api/db-instances/useUpdateDbInstance', () => ({
  useUpdateDbInstanceWithConflictRetry: () => ({ mutate }),
}));

const makeNumber = (
  path: string,
  label: string,
  validation?: Component['validation']
): Component =>
  ({
    uiType: FieldType.Number,
    path,
    fieldParams: { label },
    ...(validation ? { validation } : {}),
  }) as Component;

describe('SectionEditModal CEL validation', () => {
  it('revalidates config servers CEL when number of nodes changes', async () => {
    const sections: Record<string, Section> = {
      resources: {
        label: 'Resources',
        components: {
          numberOfNodes: makeNumber(
            'spec.components.engine.replicas',
            'Number of nodes',
            {
              required: true,
              min: 1,
              int: true,
              modes: {
                [FormMode.Edit]: {
                  celExpressions: [
                    {
                      celExpr:
                        '!(spec.components.engine.replicas == 1 && original.spec.components.engine.replicas > 1)',
                      message: 'Cannot scale down to a single node',
                    },
                  ],
                },
              },
            }
          ),
          configServers: makeNumber(
            'spec.components.configServer.replicas',
            'Nº of configuration servers',
            {
              required: true,
              min: 1,
              int: true,
              celExpressions: [
                {
                  celExpr:
                    '!(spec.components.engine.replicas > 1 && spec.components.configServer.replicas == 1)',
                  message:
                    'The number of configuration servers cannot be 1 if the number of database nodes is greater than 1',
                },
              ],
            }
          ),
        },
      },
    };

    const instance = {
      metadata: {
        name: 'test-db',
        namespace: 'ns',
      },
      spec: {
        components: {
          engine: { replicas: 3 },
          configServer: { replicas: 3 },
        },
      },
    } as unknown as Instance;

    render(
      <TestWrapper>
        <SectionEditModal
          sectionKey="resources"
          sections={sections}
          instance={instance}
          provider={{ spec: {} } as Provider}
          namespace="ns"
          onClose={vi.fn()}
          onSuccess={vi.fn()}
        />
      </TestWrapper>
    );

    const configServersInput = screen.getByLabelText(
      'Nº of configuration servers'
    );
    const numberOfNodesInput = screen.getByLabelText('Number of nodes');
    const saveButton = screen.getByTestId('form-dialog-save');

    fireEvent.change(configServersInput, { target: { value: '1' } });

    await waitFor(() => {
      expect(
        screen.getByText(
          'The number of configuration servers cannot be 1 if the number of database nodes is greater than 1'
        )
      ).toBeInTheDocument();
    });
    expect(saveButton).toBeDisabled();

    fireEvent.change(numberOfNodesInput, { target: { value: '1' } });

    await waitFor(() => {
      expect(
        screen.queryByText(
          'The number of configuration servers cannot be 1 if the number of database nodes is greater than 1'
        )
      ).not.toBeInTheDocument();
    });

    await waitFor(() => {
      expect(
        screen.getByText('Cannot scale down to a single node')
      ).toBeInTheDocument();
    });
    expect(saveButton).toBeDisabled();
  });

  it('compares quantity fields with original in the same unit so Save stays enabled', async () => {
    mutate.mockClear();

    const sections: Record<string, Section> = {
      resources: {
        label: 'Resources',
        components: {
          cpu: {
            uiType: FieldType.Quantity,
            path: 'spec.components.engine.resources.cpu',
            fieldParams: { label: 'CPU', unitLabel: 'cores' },
            validation: { required: true, min: 0.1 },
          },
          size: {
            uiType: FieldType.Quantity,
            path: 'spec.components.engine.storage.size',
            fieldParams: { label: 'Disk', unit: 'Gi' },
            validation: {
              required: true,
              min: 1,
              modes: {
                [FormMode.Edit]: {
                  celExpressions: [
                    {
                      celExpr:
                        'spec.components.engine.storage.size >= original.spec.components.engine.storage.size',
                      message: 'Disk size cannot be decreased',
                    },
                  ],
                },
              },
            },
          },
        },
      },
    };

    const instance = {
      metadata: { name: 'test-db', namespace: 'ns' },
      spec: {
        topology: { type: 'replicaSet' },
        components: {
          engine: {
            resources: { cpu: '500m' },
            storage: { size: '10240Mi' },
          },
        },
      },
    } as unknown as Instance;

    render(
      <TestWrapper>
        <SectionEditModal
          sectionKey="resources"
          sections={sections}
          instance={instance}
          provider={{ spec: {} } as Provider}
          namespace="ns"
          onClose={vi.fn()}
          onSuccess={vi.fn()}
        />
      </TestWrapper>
    );

    const cpuInput = screen.getByLabelText('CPU');
    const saveButton = screen.getByTestId('form-dialog-save');

    fireEvent.change(cpuInput, { target: { value: '1' } });

    await waitFor(() => {
      expect(saveButton).toBeEnabled();
    });
    expect(screen.queryByText(/CEL expression error/i)).not.toBeInTheDocument();
    expect(
      screen.queryByText('Disk size cannot be decreased')
    ).not.toBeInTheDocument();

    fireEvent.click(saveButton);

    await waitFor(() => expect(mutate).toHaveBeenCalled());
    expect(mutate.mock.calls[0][0].spec).toMatchObject({
      components: {
        engine: {
          resources: { cpu: '1' },
          storage: { size: '10240Mi' },
        },
      },
    });
  });
});

describe('SectionEditModal toggleable group', () => {
  it('deletes the saved values of a section switched off on edit', async () => {
    mutate.mockClear();
    const { sections } = preprocessSchema({
      replicaSet: {
        sections: {
          advanced: {
            label: 'Advanced',
            components: {
              monitoring: {
                uiType: 'group',
                groupType: GroupType.Toggleable,
                label: 'Monitoring',
                components: {
                  interval: makeNumber('spec.monitoring.interval', 'Interval'),
                },
              },
            },
          },
        },
      },
    }).replicaSet;

    const instance = {
      metadata: { name: 'test-db', namespace: 'ns' },
      spec: { monitoring: { interval: 30 }, other: 'kept' },
    } as unknown as Instance;

    render(
      <TestWrapper>
        <SectionEditModal
          sectionKey="advanced"
          sections={sections}
          instance={instance}
          provider={{ spec: {} } as Provider}
          namespace="ns"
          onClose={vi.fn()}
          onSuccess={vi.fn()}
        />
      </TestWrapper>
    );

    // A saved value means the section loads on.
    const toggle = screen.getByRole('switch', { name: 'Enable Monitoring' });
    expect(toggle).toBeChecked();

    fireEvent.click(toggle);
    const saveButton = screen.getByTestId('form-dialog-save');
    await waitFor(() => expect(saveButton).toBeEnabled());
    fireEvent.click(saveButton);

    await waitFor(() => expect(mutate).toHaveBeenCalled());
    // No `monitoring: {}` left behind either.
    expect(mutate.mock.calls[0][0].spec).toEqual({ other: 'kept' });
  });
});

describe('SectionEditModal dataSource field disabled on edit', () => {
  const { sections } = preprocessSchema({
    replicaSet: {
      sections: {
        advanced: {
          label: 'Advanced',
          components: {
            storageClass: {
              uiType: FieldType.Select,
              path: 'spec.storageClass',
              fieldParams: {
                label: 'Storage class',
                modes: { [FormMode.Edit]: { disabled: true } },
              },
              dataSource: { provider: 'storageClasses' },
            },
            interval: makeNumber('spec.interval', 'Interval'),
          },
        },
      },
    },
  }).replicaSet;

  const renderModal = (spec: Record<string, unknown>) => {
    const instance = {
      metadata: { name: 'test-db', namespace: 'ns' },
      spec,
    } as unknown as Instance;

    render(
      <TestWrapper>
        <SectionEditModal
          sectionKey="advanced"
          sections={sections}
          instance={instance}
          provider={{ spec: {} } as Provider}
          namespace="ns"
          onClose={vi.fn()}
          onSuccess={vi.fn()}
        />
      </TestWrapper>
    );
  };

  beforeEach(() => {
    mutate.mockClear();
    clusterInfo.data = { storageClassNames: ['standard'] };
  });

  afterEach(() => {
    clusterInfo.data = undefined;
    vi.restoreAllMocks();
  });

  it.each([
    ['unset', {}],
    ['missing from the options', { storageClass: 'deleted-class' }],
  ])(
    'saves another field without touching a storage class that is %s',
    async (_case, storedStorageClass) => {
      renderModal({ interval: 30, ...storedStorageClass });

      fireEvent.change(screen.getByLabelText('Interval'), {
        target: { value: '60' },
      });
      const saveButton = screen.getByTestId('form-dialog-save');
      await waitFor(() => expect(saveButton).toBeEnabled());
      fireEvent.click(saveButton);

      await waitFor(() => expect(mutate).toHaveBeenCalled());
      expect(mutate.mock.calls[0][0].spec).toEqual({
        interval: 60,
        ...storedStorageClass,
      });
    }
  );

  it('shows a stored storage class missing from the options as unavailable', () => {
    const warn = vi.spyOn(console, 'warn');

    renderModal({ interval: 30, storageClass: 'deleted-class' });

    expect(
      screen.getByText(Messages.unavailableOption('deleted-class'))
    ).toBeInTheDocument();
    expect(
      screen.getByText(Messages.unavailableLockedValueHint)
    ).toBeInTheDocument();
    expect(warn).not.toHaveBeenCalledWith(
      expect.stringContaining('out-of-range')
    );
  });

  it('explains what an unset storage class means', () => {
    renderModal({ interval: 30 });

    expect(
      screen.getByText(ProvidersMessages.storageClassesUnsetValueHint)
    ).toBeInTheDocument();
  });
});

describe('SectionEditModal while the database converges', () => {
  const { sections } = preprocessSchema({
    replicaSet: {
      sections: {
        resources: {
          label: 'Resources',
          components: {
            replicas: makeNumber('spec.components.engine.replicas', 'Nodes'),
          },
        },
      },
    },
  }).replicaSet;

  const renderModal = (reconciling: 'True' | 'False') =>
    render(
      <TestWrapper>
        <SectionEditModal
          sectionKey="resources"
          sections={sections}
          instance={
            {
              metadata: { name: 'test-db', namespace: 'ns' },
              spec: { components: { engine: { replicas: 3 } } },
              status: {
                conditions: [
                  {
                    type: 'Reconciling',
                    status: reconciling,
                    reason: 'Upgrading',
                    message: '',
                    lastTransitionTime: '2026-03-01T11:56:00Z',
                  },
                ],
              },
            } as unknown as Instance
          }
          provider={{ spec: {} } as Provider}
          namespace="ns"
          onClose={vi.fn()}
          onSuccess={vi.fn()}
        />
      </TestWrapper>
    );

  it('says that a change is still being applied, and lets the person edit', () => {
    renderModal('True');

    expect(screen.getByTestId('edit-reconciling-hint')).toBeInTheDocument();
    expect(screen.getByLabelText('Nodes')).toBeEnabled();
  });

  it('says nothing once the database has converged', () => {
    renderModal('False');

    expect(
      screen.queryByTestId('edit-reconciling-hint')
    ).not.toBeInTheDocument();
  });
});
