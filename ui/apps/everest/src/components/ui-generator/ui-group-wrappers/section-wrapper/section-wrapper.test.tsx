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
import { describe, it, expect } from 'vitest';
import SectionWrapper from './section-wrapper';

describe('SectionWrapper', () => {
  it('renders children', () => {
    render(
      <SectionWrapper>
        <span data-testid="child">content</span>
      </SectionWrapper>
    );
    expect(screen.getByTestId('child')).toBeInTheDocument();
  });

  it('renders label as section heading', () => {
    render(<SectionWrapper label="Storage">content</SectionWrapper>);
    expect(screen.getByText('Storage')).toBeInTheDocument();
  });

  it('renders description when provided', () => {
    render(
      <SectionWrapper label="Storage" description="Defines storage type">
        content
      </SectionWrapper>
    );
    expect(screen.getByText('Defines storage type')).toBeInTheDocument();
  });

  it('does not render description element when description is omitted', () => {
    render(<SectionWrapper label="Storage">content</SectionWrapper>);
    expect(screen.queryByText('Defines storage type')).not.toBeInTheDocument();
  });

  it('applies disabled styling when disabled is true', () => {
    const { container } = render(
      <SectionWrapper label="Storage" disabled>
        content
      </SectionWrapper>
    );
    const box = container.querySelector('.percona-rounded-box');
    expect(box).toBeInTheDocument();
    expect(box).toHaveStyle({ opacity: '0.5', pointerEvents: 'none' });
  });

  it('renders without label or description', () => {
    render(
      <SectionWrapper>
        <span data-testid="child">content</span>
      </SectionWrapper>
    );
    expect(screen.getByTestId('child')).toBeInTheDocument();
  });
});
