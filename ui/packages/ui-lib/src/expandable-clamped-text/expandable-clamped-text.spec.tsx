import { fireEvent, render, screen } from '@testing-library/react';
import ExpandableClampedText from './expandable-clamped-text';

let hasOverflow = false;

class ResizeObserverMock {
  observe() {}
  disconnect() {}
}

beforeAll(() => {
  Object.defineProperty(globalThis, 'ResizeObserver', {
    value: ResizeObserverMock,
    writable: true,
    configurable: true,
  });

  Object.defineProperty(HTMLElement.prototype, 'scrollHeight', {
    configurable: true,
    get() {
      return hasOverflow ? 100 : 40;
    },
  });

  Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
    configurable: true,
    get() {
      return 40;
    },
  });
});

afterAll(() => {
  delete (globalThis as { ResizeObserver?: unknown }).ResizeObserver;
});

it('should not render toggle when text does not overflow', () => {
  hasOverflow = false;

  render(<ExpandableClampedText value="Short value" />);

  expect(screen.queryByTestId('expandable-clamped-text-toggle')).toBeNull();
});

it('should expand inline when using inline strategy', () => {
  hasOverflow = true;

  render(
    <ExpandableClampedText
      value="Long value"
      dataTestId="inline-text"
      expandStrategy={{ type: 'inline' }}
    />
  );

  const toggle = screen.getByTestId('inline-text-toggle');
  expect(toggle).toHaveTextContent('Show more');

  fireEvent.click(toggle);

  expect(screen.getByTestId('inline-text-toggle')).toHaveTextContent(
    'Show less'
  );
});

it('should open dialog when using dialog strategy', () => {
  hasOverflow = true;

  render(
    <ExpandableClampedText
      value="Long value"
      dataTestId="dialog-text"
      expandStrategy={{ type: 'dialog', dialogTitle: 'Config value' }}
    />
  );

  fireEvent.click(screen.getByTestId('dialog-text-toggle'));

  expect(screen.getByText('Config value')).toBeInTheDocument();
  expect(screen.getByTestId('dialog-text-dialog-close')).toBeInTheDocument();
});

it('should call callback when using navigate strategy', () => {
  hasOverflow = true;
  const onExpand = vi.fn();

  render(
    <ExpandableClampedText
      value="Long value"
      dataTestId="navigate-text"
      expandStrategy={{ type: 'navigate', onExpand }}
    />
  );

  fireEvent.click(screen.getByTestId('navigate-text-toggle'));

  expect(onExpand).toHaveBeenCalledTimes(1);
});

it('should not render toggle for scroll strategy', () => {
  hasOverflow = true;

  render(
    <ExpandableClampedText
      value="Long value"
      dataTestId="scroll-text"
      expandStrategy={{ type: 'scroll', maxHeight: 80 }}
    />
  );

  expect(screen.queryByTestId('scroll-text-toggle')).toBeNull();
});

it('should close dialog when clicking the close button', () => {
  hasOverflow = true;

  render(
    <ExpandableClampedText
      value="Long value"
      dataTestId="dialog-close-test"
      expandStrategy={{ type: 'dialog', dialogTitle: 'Details' }}
    />
  );

  const toggle = screen.getByTestId('dialog-close-test-toggle');
  fireEvent.click(toggle);
  expect(toggle).toHaveAttribute('aria-expanded', 'true');

  fireEvent.click(screen.getByTestId('dialog-close-test-dialog-close'));
  expect(toggle).toHaveAttribute('aria-expanded', 'false');
});

it('should reset expanded state when value changes', () => {
  hasOverflow = true;

  const { rerender } = render(
    <ExpandableClampedText
      value="First value"
      dataTestId="reset-test"
      expandStrategy={{ type: 'inline' }}
    />
  );

  const toggle = screen.getByTestId('reset-test-toggle');
  fireEvent.click(toggle);
  expect(toggle).toHaveTextContent('Show less');

  rerender(
    <ExpandableClampedText
      value="Second value"
      dataTestId="reset-test"
      expandStrategy={{ type: 'inline' }}
    />
  );

  expect(screen.getByTestId('reset-test-toggle')).toHaveTextContent(
    'Show more'
  );
});

it('should handle empty string without toggle', () => {
  hasOverflow = false;

  render(<ExpandableClampedText value="" dataTestId="empty-test" />);

  expect(screen.getByTestId('empty-test-value')).toHaveTextContent('');
  expect(screen.queryByTestId('empty-test-toggle')).toBeNull();
});
