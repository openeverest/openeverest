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
import { TestWrapper } from 'utils/test';
import SwitchGroupWrapper from './switch-group-wrapper';

const renderWrapper = (defaultExpanded?: boolean) =>
  render(
    <SwitchGroupWrapper
      label="Customize resource requests"
      labelCaption="Lower requests below the limits."
      defaultExpanded={defaultExpanded}
    >
      <div>child content</div>
    </SwitchGroupWrapper>,
    { wrapper: TestWrapper }
  );

const getSwitch = () => screen.getByRole('switch');

describe('SwitchGroupWrapper', () => {
  it('renders collapsed with the switch off by default', () => {
    renderWrapper();
    expect(getSwitch()).not.toBeChecked();
    expect(screen.getByText('child content')).not.toBeVisible();
  });

  it('reveals children when the switch is turned on', () => {
    renderWrapper();
    fireEvent.click(getSwitch());
    expect(getSwitch()).toBeChecked();
    expect(screen.getByText('child content')).toBeVisible();
  });

  it('renders expanded when defaultExpanded is true', () => {
    renderWrapper(true);
    expect(getSwitch()).toBeChecked();
    expect(screen.getByText('child content')).toBeVisible();
  });
});
