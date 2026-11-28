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
import {
  WidgetComponent,
  WIDGET_UI_TYPE,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';
import { PodSchedulingPolicySection } from './pod-scheduling-policy-section';
import { AffinityOperator } from 'shared-types/affinity.types';

vi.mock('components/ui-generator/ui-generator', () => ({
  UIGenerator: () => null,
}));

const affinity = {
  nodeAffinity: {
    requiredDuringSchedulingIgnoredDuringExecution: {
      nodeSelectorTerms: [
        {
          matchExpressions: [
            { key: 'gpu', operator: AffinityOperator.In, values: ['true'] },
          ],
        },
      ],
    },
  },
};

const item: WidgetComponent = {
  uiType: WIDGET_UI_TYPE,
  widgetType: WidgetType.PodSchedulingPolicy,
  id: 'podSchedulingPolicy',
  _widgetTargets: [
    {
      key: 'engine',
      path: "spec.components.engine.schedulingPolicy['nvidia.com/gpu']",
    },
    {
      key: 'proxy',
      path: 'spec.components.proxy.schedulingPolicy.affinity',
    },
  ],
};

const Form = () => {
  const methods = useForm({
    defaultValues: {
      spec: {
        components: {
          engine: {
            schedulingPolicy: {
              'nvidia%2Ecom%2Fgpu': affinity,
            },
          },
          proxy: {
            schedulingPolicy: { affinity },
          },
        },
      },
    },
  });

  return (
    <FormProvider {...methods}>
      <PodSchedulingPolicySection name="g-podSchedulingPolicy" item={item} />
    </FormProvider>
  );
};

describe('PodSchedulingPolicySection', () => {
  it('watches ordinary and qualified target paths and preserves tab counts', () => {
    render(
      <TestWrapper>
        <Form />
      </TestWrapper>
    );

    expect(
      screen.getByTestId('scheduling-component-tab-engine')
    ).toHaveTextContent('engine (1)');
    expect(
      screen.getByTestId('scheduling-component-tab-proxy')
    ).toHaveTextContent('proxy (1)');
  });
});
