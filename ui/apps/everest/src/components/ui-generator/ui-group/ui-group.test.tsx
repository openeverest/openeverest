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
import { TestWrapper } from 'utils/test';
import {
  ComponentGroup,
  GroupType,
} from 'components/ui-generator/ui-generator.types';
import UIGroup from './ui-group';

const group = (overrides: Partial<ComponentGroup> = {}): ComponentGroup => ({
  uiType: 'group',
  components: {},
  ...overrides,
});

const renderGroup = (groupType?: GroupType, item?: ComponentGroup) =>
  render(
    <UIGroup groupType={groupType} item={item}>
      <div>group-child</div>
    </UIGroup>,
    { wrapper: TestWrapper }
  );

describe('UIGroup dispatch', () => {
  it('renders a bordered card and forwards label + description', () => {
    const { container } = renderGroup(
      GroupType.Bordered,
      group({ label: 'Advanced', description: 'More options' })
    );
    expect(screen.getByText('group-child')).toBeInTheDocument();
    expect(screen.getByText('Advanced')).toBeInTheDocument();
    expect(screen.getByText('More options')).toBeInTheDocument();
    expect(container.querySelector('.percona-rounded-box')).toBeInTheDocument();
  });

  it('still renders the accordion group with its label (regression)', () => {
    renderGroup(GroupType.Accordion, group({ label: 'Section' }));
    expect(screen.getByText('Section')).toBeInTheDocument();
    expect(screen.getByText('group-child')).toBeInTheDocument();
  });

  it('still renders the line group children (regression)', () => {
    renderGroup(GroupType.Line, group());
    expect(screen.getByText('group-child')).toBeInTheDocument();
  });

  it('falls back to a plain stack when no groupType is set', () => {
    renderGroup(undefined, group());
    expect(screen.getByText('group-child')).toBeInTheDocument();
  });
});
