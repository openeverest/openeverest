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

import { render, screen, fireEvent } from '@testing-library/react';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import { Affinity, AffinityOperator } from 'shared-types/affinity.types';
import {
  WidgetComponent,
  WIDGET_UI_TYPE,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';
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
});
