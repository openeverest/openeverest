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

import { fireEvent, render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
  AffinityOperator,
  AffinityPriority,
  AffinityType,
} from 'shared-types/affinity.types';
import { AffinityGroup } from '../affinity-group.types';
import { AffinityGroupList } from './affinity-group-list';

const group = (
  type: AffinityType,
  priority: AffinityPriority,
  key: string,
  weight?: number
): AffinityGroup => ({
  type,
  priority,
  weight,
  topologyKey:
    type === AffinityType.NodeAffinity ? undefined : 'kubernetes.io/hostname',
  conditions: [{ key, operator: AffinityOperator.Exists }],
});

// Stored order as the converter emits it: preferred node terms come first.
const groups = [
  group(AffinityType.NodeAffinity, AffinityPriority.Preferred, 'light', 10),
  group(AffinityType.NodeAffinity, AffinityPriority.Preferred, 'heavy', 80),
  group(AffinityType.PodAntiAffinity, AffinityPriority.Required, 'anti'),
  group(AffinityType.NodeAffinity, AffinityPriority.Required, 'zone-a'),
  group(AffinityType.NodeAffinity, AffinityPriority.Required, 'zone-b'),
];

const renderedKeys = () =>
  screen
    .getAllByTestId(/^editable-item$/)
    .map(
      (item) =>
        within(item).getByText(/^(light|heavy|anti|zone-a|zone-b)$/).textContent
    );

describe('AffinityGroupList', () => {
  it('orders required before preferred, node first and preferences by weight', () => {
    render(<AffinityGroupList groups={groups} />);

    expect(renderedKeys()).toEqual([
      'zone-a',
      'zone-b',
      'anti',
      'heavy',
      'light',
    ]);
  });

  it('binds only required node affinity groups as OR alternatives', () => {
    render(<AffinityGroupList groups={groups} />);

    const orGroup = screen.getByTestId('affinity-or-group');
    expect(within(orGroup).getByText('zone-a')).toBeInTheDocument();
    expect(within(orGroup).getByText('zone-b')).toBeInTheDocument();
    expect(within(orGroup).queryByText('anti')).not.toBeInTheDocument();
    expect(screen.getAllByText('OR')).toHaveLength(1);
  });

  it('edits and removes by stored position, not displayed position', () => {
    const onEdit = vi.fn();
    const onRemove = vi.fn();
    render(
      <AffinityGroupList
        groups={groups}
        actions={{ onAdd: vi.fn(), onEdit, onRemove }}
      />
    );

    fireEvent.click(
      screen.getByTestId('edit-editable-item-button-affinity-group-3')
    );
    fireEvent.click(
      screen.getByTestId('delete-editable-item-button-affinity-group-1')
    );

    expect(onEdit).toHaveBeenCalledWith(3);
    expect(onRemove).toHaveBeenCalledWith(1);
  });
});
