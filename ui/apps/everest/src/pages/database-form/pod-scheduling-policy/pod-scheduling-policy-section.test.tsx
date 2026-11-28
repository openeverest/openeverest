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
      <PodSchedulingPolicySection item={item} />
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
