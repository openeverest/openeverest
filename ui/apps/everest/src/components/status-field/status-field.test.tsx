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
import StatusField from './status-field';
import { BaseStatus } from './status-field.types';
import { STATUS_TO_ICON } from './status-field.utils';

describe('STATUS_TO_ICON', () => {
  it.each(Object.keys(STATUS_TO_ICON) as BaseStatus[])(
    'defines an icon for "%s"',
    (status) => {
      expect(STATUS_TO_ICON[status]).toBeDefined();
    }
  );
});

describe('StatusField', () => {
  const statusMap = { active: 'success' as BaseStatus };

  it('uses dataTestId as a prefix for the container testid', () => {
    render(
      <TestWrapper>
        <StatusField
          status="active"
          statusMap={statusMap}
          dataTestId="db-cluster"
        />
      </TestWrapper>
    );
    expect(screen.getByTestId('db-cluster-status')).toBeInTheDocument();
  });

  it('uses "status" as the container testid when dataTestId is omitted', () => {
    render(
      <TestWrapper>
        <StatusField status="active" statusMap={statusMap} />
      </TestWrapper>
    );
    expect(screen.getByTestId('status')).toBeInTheDocument();
  });

  it('renders children inside the container', () => {
    render(
      <TestWrapper>
        <StatusField status="active" statusMap={statusMap}>
          <span>Active</span>
        </StatusField>
      </TestWrapper>
    );
    expect(screen.getByText('Active')).toBeInTheDocument();
  });

  it('falls back to defaultIcon when the mapped status is not in STATUS_TO_ICON', () => {
    const FallbackIcon = () => <svg data-testid="fallback-icon" />;
    render(
      <TestWrapper>
        <StatusField
          status="active"
          statusMap={{ active: 'not-a-real-status' as BaseStatus }}
          defaultIcon={FallbackIcon}
        />
      </TestWrapper>
    );
    expect(screen.getByTestId('fallback-icon')).toBeInTheDocument();
  });
});
