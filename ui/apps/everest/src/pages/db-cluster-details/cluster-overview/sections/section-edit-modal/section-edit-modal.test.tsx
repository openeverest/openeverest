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
} from 'components/ui-generator/ui-generator.types';
import type { Instance, Provider } from 'shared-types/api.types';
import SectionEditModal from './section-edit-modal';

vi.mock('hooks/api/kubernetesClusters/useKubernetesClusterInfo', () => ({
  useKubernetesClusterInfo: () => ({ data: undefined }),
}));

vi.mock('hooks/api/db-instances/useUpdateDbInstance', () => ({
  useUpdateDbInstanceWithConflictRetry: () => ({ mutate: vi.fn() }),
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
});
