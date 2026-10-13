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
import { describe, expect, it } from 'vitest';
import { AffinityOperator } from 'shared-types/affinity.types';
import {
  WidgetComponent,
  WIDGET_UI_TYPE,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';
import { SchedulingPolicyView } from './scheduling-policy-view';

const item: WidgetComponent = {
  uiType: WIDGET_UI_TYPE,
  widgetType: WidgetType.PodSchedulingPolicy,
  id: 'podSchedulingPolicy',
};

const proxyAffinity = {
  nodeAffinity: {
    preferredDuringSchedulingIgnoredDuringExecution: [
      {
        weight: 10,
        preference: {
          matchExpressions: [
            { key: 'disktype', operator: AffinityOperator.In, values: ['ssd'] },
          ],
        },
      },
    ],
    requiredDuringSchedulingIgnoredDuringExecution: {
      nodeSelectorTerms: [
        {
          matchExpressions: [
            { key: 'zone', operator: AffinityOperator.In, values: ['us-west'] },
          ],
        },
      ],
    },
  },
};

describe('SchedulingPolicyView', () => {
  it('opens on the first component with rules and offers no editing', () => {
    render(
      <SchedulingPolicyView
        item={item}
        value={{ engine: undefined, proxy: proxyAffinity }}
      />
    );

    expect(
      screen.getByTestId('scheduling-component-tab-proxy')
    ).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByText('proxy (2)')).toBeInTheDocument();
    expect(screen.getByText('zone')).toBeInTheDocument();
    expect(screen.getByText('disktype')).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /add rule group/i })
    ).toBeNull();
    expect(
      screen.queryByTestId('edit-editable-item-button-affinity-group-0')
    ).toBeNull();
    expect(
      screen.queryByTestId('delete-editable-item-button-affinity-group-0')
    ).toBeNull();
  });

  it('shows a read-only empty state on a component without rules', () => {
    render(
      <SchedulingPolicyView
        item={item}
        value={{ engine: undefined, proxy: proxyAffinity }}
      />
    );

    fireEvent.click(screen.getByTestId('scheduling-component-tab-engine'));

    expect(
      screen.getByText('No affinity rules for this component.')
    ).toBeInTheDocument();
  });

  it('collapses to a single line when no component has rules', () => {
    render(
      <SchedulingPolicyView item={item} value={{ engine: {}, proxy: {} }} />
    );

    expect(
      screen.getByText('No pod scheduling rules configured.')
    ).toBeInTheDocument();
    expect(screen.queryByRole('tab')).toBeNull();
  });
});
