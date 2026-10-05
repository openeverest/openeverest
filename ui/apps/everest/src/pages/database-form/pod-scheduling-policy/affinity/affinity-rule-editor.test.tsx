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

import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import { Affinity, AffinityOperator } from 'shared-types/affinity.types';
import {
  WidgetComponent,
  WIDGET_UI_TYPE,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';
import { SchedulingComponentContext } from '../scheduling-component-context';
import { AffinityRuleEditor } from './affinity-rule-editor';

const item: WidgetComponent = {
  uiType: WIDGET_UI_TYPE,
  widgetType: WidgetType.Affinity,
  path: 'spec.affinity',
};

const nodeAffinity: Affinity = {
  nodeAffinity: {
    requiredDuringSchedulingIgnoredDuringExecution: {
      nodeSelectorTerms: [
        {
          matchExpressions: [
            { key: 'disktype', operator: AffinityOperator.In, values: ['ssd'] },
          ],
        },
      ],
    },
  },
};

const AffinityProbe = () => {
  const value = useWatch({ name: 'spec.affinity' });
  return <div data-testid="probe">{JSON.stringify(value ?? {})}</div>;
};

const readProbe = () =>
  JSON.parse(screen.getByTestId('probe').textContent ?? '{}');

// Built outside the UI (kubectl / GitOps): an empty selector matching all pods
// next to a node rule, plus term fields the editor has no controls for.
const antiAffinityTerm = (labelSelector: Record<string, unknown>) => ({
  topologyKey: 'kubernetes.io/hostname',
  labelSelector,
});
const withAntiAffinity = (
  labelSelector: Record<string, unknown>
): Affinity => ({
  ...nodeAffinity,
  podAntiAffinity: {
    requiredDuringSchedulingIgnoredDuringExecution: [
      antiAffinityTerm(labelSelector),
    ],
  },
});
const richPodAffinity = {
  podAffinity: {
    requiredDuringSchedulingIgnoredDuringExecution: [
      {
        topologyKey: 'kubernetes.io/hostname',
        namespaces: ['db'],
        namespaceSelector: {},
        labelSelector: {
          matchLabels: { app: 'mysql' },
          matchExpressions: [
            { key: 'tier', operator: AffinityOperator.Exists },
          ],
        },
      },
    ],
  },
};

const saveGroupUnchanged = async (index: number) => {
  fireEvent.click(
    screen.getByTestId(`edit-editable-item-button-affinity-group-${index}`)
  );
  const save = screen.getByTestId('form-dialog-save');
  await waitFor(() => expect(save).toBeEnabled());
  fireEvent.click(save);
  await waitFor(() =>
    expect(screen.queryByTestId('form-dialog-save')).not.toBeInTheDocument()
  );
};

const Wrapper = ({ initial }: { initial?: Affinity }) => {
  const methods = useForm({ defaultValues: { spec: { affinity: initial } } });
  return (
    <FormProvider {...methods}>
      <AffinityRuleEditor name="spec.affinity" item={item} />
      <AffinityProbe />
    </FormProvider>
  );
};

describe('AffinityRuleEditor', () => {
  it('renders a group derived from the stored k8s Affinity', () => {
    render(<Wrapper initial={nodeAffinity} />);
    expect(screen.getByText('disktype')).toBeInTheDocument();
    expect(screen.getByText('[ssd]')).toBeInTheDocument();
  });

  it('deletes a group and writes the reduced Affinity back to the form', () => {
    render(<Wrapper initial={nodeAffinity} />);
    expect(screen.getByTestId('probe')).toHaveTextContent('nodeAffinity');

    fireEvent.click(
      screen.getByTestId('delete-editable-item-button-affinity-group-0')
    );

    expect(screen.getByTestId('probe')).toHaveTextContent('{}');
  });

  it('shows the empty state when there are no rules', () => {
    render(<Wrapper initial={{}} />);
    expect(screen.getByText(/No affinity rules yet/i)).toBeInTheDocument();
  });

  describe('empty label selector (matches all pods)', () => {
    it('survives deleting another group of the component', () => {
      render(<Wrapper initial={withAntiAffinity({})} />);

      fireEvent.click(
        screen.getByTestId('delete-editable-item-button-affinity-group-0')
      );

      expect(readProbe()).toStrictEqual({
        podAntiAffinity: {
          requiredDuringSchedulingIgnoredDuringExecution: [
            antiAffinityTerm({}),
          ],
        },
      });
    });

    it('survives saving its own group through the dialog', async () => {
      render(<Wrapper initial={withAntiAffinity({})} />);

      await saveGroupUnchanged(1);

      expect(readProbe()).toStrictEqual(withAntiAffinity({}));
    });

    it('is reported as matching all pods, not as matching none', () => {
      render(<Wrapper initial={withAntiAffinity({})} />);
      expect(screen.getByText(/matches all pods/i)).toBeInTheDocument();
      expect(screen.queryByText(/matches no pods/i)).not.toBeInTheDocument();
    });
  });

  it('reports a pod rule without a label selector as matching no pods', () => {
    render(
      <Wrapper
        initial={{
          podAntiAffinity: {
            requiredDuringSchedulingIgnoredDuringExecution: [
              { topologyKey: 'kubernetes.io/hostname' },
            ],
          },
        }}
      />
    );
    expect(screen.getByText(/matches no pods/i)).toBeInTheDocument();
  });

  it('keeps term fields the editor does not model after a dialog save', async () => {
    render(<Wrapper initial={richPodAffinity} />);

    await saveGroupUnchanged(0);

    expect(readProbe()).toStrictEqual(richPodAffinity);
  });

  it('names the component of the active scheduling tab in the dialog', () => {
    render(
      <SchedulingComponentContext.Provider value={{ component: 'proxy' }}>
        <Wrapper initial={{}} />
      </SchedulingComponentContext.Provider>
    );
    fireEvent.click(screen.getByRole('button', { name: 'Add rule group' }));
    expect(
      screen.getByText('Add rule group to the proxy component')
    ).toBeInTheDocument();
  });
});
