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
import {
  FieldType,
  type Component,
  type Section,
} from 'components/ui-generator/ui-generator.types';
import { Messages as CelMessages } from 'components/ui-generator/utils/schema-builder/cel-validation/cel-validation.messages';
import type { Instance, Provider } from 'shared-types/api.types';
import SectionEditModal from '../section-edit-modal';
import { Messages } from '../section-edit-modal.messages';

const { mutate, storageClasses } = vi.hoisted(() => ({
  mutate: vi.fn(),
  storageClasses: { data: [] },
}));

vi.mock('hooks/api/storage-classes/useStorageClasses', () => ({
  useStorageClassesList: () => storageClasses,
}));

vi.mock('hooks/api/db-instances/useUpdateDbInstance', () => ({
  useUpdateDbInstanceWithConflictRetry: () => ({ mutate }),
}));

const ALERT = 'section-outside-field-errors';
const NODES_PATH = 'spec.components.engine.replicas';
const PROXIES_PATH = 'spec.components.proxy.replicas';

const makeField = (
  uiType: FieldType.Number | FieldType.Text,
  path: string,
  label: string,
  validation?: Component['validation']
): Component => ({
  uiType,
  path,
  fieldParams: { label },
  ...(validation ? { validation } : {}),
});

const makeSections = (nodesRule: {
  celExpr: string;
  message?: string;
}): Record<string, Section> => ({
  resources: {
    label: 'Resources',
    components: {
      nodes: makeField(FieldType.Number, NODES_PATH, 'Number of nodes', {
        celExpressions: [nodesRule],
      }),
    },
  },
  proxy: {
    label: 'Proxy',
    components: {
      proxies: makeField(FieldType.Number, PROXIES_PATH, 'Number of proxies'),
    },
  },
});

const makeInstance = (spec: Record<string, unknown>): Instance => ({
  metadata: { name: 'test-db', namespace: 'ns' },
  spec: { providerRef: { name: 'psmdb' }, ...spec },
});

const provider: Provider = { spec: {} };

const renderModal = (
  sectionKey: string,
  sections: Record<string, Section>,
  instance: Instance
) =>
  render(
    <TestWrapper>
      <SectionEditModal
        sectionKey={sectionKey}
        sections={sections}
        instance={instance}
        provider={provider}
        namespace="ns"
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />
    </TestWrapper>
  );

describe('SectionEditModal errors outside the dialog', () => {
  it('names the field, its section and the rule that keeps Save disabled, until the rule passes', async () => {
    const message = 'Nodes cannot be fewer than proxies';
    renderModal(
      'proxy',
      makeSections({ celExpr: `${NODES_PATH} >= ${PROXIES_PATH}`, message }),
      makeInstance({
        components: { engine: { replicas: 1 }, proxy: { replicas: 3 } },
      })
    );

    const saveButton = screen.getByTestId('form-dialog-save');
    const alert = await screen.findByTestId(ALERT);

    expect(alert).toHaveTextContent(Messages.outsideErrorsTitle);
    expect(alert).toHaveTextContent(`Resources › Number of nodes: ${message}`);
    expect(saveButton).toBeDisabled();

    fireEvent.change(screen.getByLabelText('Number of proxies'), {
      target: { value: '1' },
    });

    await waitFor(() => {
      expect(screen.queryByTestId(ALERT)).not.toBeInTheDocument();
    });
    expect(saveButton).toBeEnabled();
  });

  it('reports a rule that cannot be evaluated as the rule failing, and logs it once', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const celExpr = 'spec.components.engine.absent.size > 0';
    renderModal(
      'proxy',
      makeSections({ celExpr, message: 'Size must be positive' }),
      makeInstance({
        components: { engine: { replicas: 1 }, proxy: { replicas: 3 } },
      })
    );

    const alert = await screen.findByTestId(ALERT);

    expect(alert).toHaveTextContent(
      `Resources › Number of nodes: ${CelMessages.ruleNotEvaluable}`
    );
    expect(alert).not.toHaveTextContent('Size must be positive');
    expect(screen.getByTestId('form-dialog-save')).toBeDisabled();

    fireEvent.change(screen.getByLabelText('Number of proxies'), {
      target: { value: '5' },
    });
    await waitFor(() => {
      expect(screen.getByLabelText('Number of proxies')).toHaveValue(5);
    });

    const ruleWarnings = warn.mock.calls.filter(
      ([text]) => typeof text === 'string' && text.includes(celExpr)
    );
    expect(ruleWarnings).toHaveLength(1);
    expect(ruleWarnings[0][0]).toContain(NODES_PATH);

    warn.mockRestore();
  });
});

describe('SectionEditModal errors on open', () => {
  it('marks an empty required field invalid before it is touched', async () => {
    renderModal(
      'tls',
      {
        tls: {
          label: 'TLS',
          components: {
            mode: makeField(
              FieldType.Text,
              'spec.parameters.tls.mode',
              'Mode',
              {
                required: true,
              }
            ),
          },
        },
      },
      makeInstance({})
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/Mode/)).toHaveAttribute(
        'aria-invalid',
        'true'
      );
    });
    expect(screen.getByTestId('form-dialog-save')).toBeDisabled();
    expect(screen.queryByTestId(ALERT)).not.toBeInTheDocument();
  });
});
