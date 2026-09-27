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
import BorderedWrapper from './bordered-wrapper';

const renderWrapper = (props: {
  label?: string;
  description?: string;
}) =>
  render(
    <BorderedWrapper {...props}>
      <div>body-child</div>
    </BorderedWrapper>,
    { wrapper: TestWrapper }
  );

describe('BorderedWrapper', () => {
  it('renders a bordered card around its children', () => {
    const { container } = renderWrapper({});
    expect(screen.getByText('body-child')).toBeInTheDocument();
    expect(container.querySelector('.percona-rounded-box')).toBeInTheDocument();
  });

  it('shows a heading with label and description when provided', () => {
    renderWrapper({ label: 'Storage', description: 'Disk settings' });
    expect(screen.getByText('Storage')).toBeInTheDocument();
    expect(screen.getByText('Disk settings')).toBeInTheDocument();
  });

  it('renders no heading text when neither label nor description is given', () => {
    renderWrapper({});
    // Only the body renders; no label/description nodes exist.
    expect(screen.getByText('body-child')).toBeInTheDocument();
    expect(screen.queryByText('Storage')).not.toBeInTheDocument();
  });
});
