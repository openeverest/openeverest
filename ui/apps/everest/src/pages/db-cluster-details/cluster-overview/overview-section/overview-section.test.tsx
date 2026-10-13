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
import { OverviewSection } from './overview-section';

describe('OverviewSection', () => {
  it('starts collapsed when asked and toggles its content with the chevron', () => {
    render(
      <OverviewSection
        title="Pod scheduling policy"
        dataTestId="policy"
        collapsible
        defaultExpanded={false}
        loading={false}
      >
        <span>rules</span>
      </OverviewSection>
    );

    const toggle = screen.getByTestId('policy-toggle');
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByText('rules')).not.toBeInTheDocument();

    fireEvent.click(toggle);

    expect(toggle).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('rules')).toBeInTheDocument();
  });
});
