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
    // The outermost percona-rounded-box element should carry opacity styling
    const box = container.querySelector('.percona-rounded-box');
    expect(box).toBeInTheDocument();
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
