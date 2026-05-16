import { screen, render, fireEvent } from '@testing-library/react';
import { TestWrapper } from 'utils/test';
import { PreviewSection, TruncatedPreviewText } from './preview-section';

describe('PreviewSection', () => {
  it('should show order number and title', () => {
    render(
      <TestWrapper>
        <PreviewSection title="My title" order={2}>
          Some text
        </PreviewSection>
      </TestWrapper>
    );

    expect(screen.getByText('2. My title')).toBeInTheDocument();
  });

  it('should not show content by default', () => {
    render(
      <TestWrapper>
        <PreviewSection title="My title" order={2}>
          Some text
        </PreviewSection>
      </TestWrapper>
    );

    expect(screen.queryByText('Some text')).not.toBeInTheDocument();
  });

  it('should not show edit icon by default', () => {
    render(
      <TestWrapper>
        <PreviewSection title="My title" order={2}>
          Some text
        </PreviewSection>
      </TestWrapper>
    );

    expect(screen.queryByTestId('edit-section-2')).not.toBeInTheDocument();
  });

  it('should show content only when hasBeenReach is true', () => {
    render(
      <TestWrapper>
        <PreviewSection title="My title" order={2} hasBeenReached>
          Some text
        </PreviewSection>
      </TestWrapper>
    );

    expect(screen.getByText('Some text')).toBeInTheDocument();
  });

  it('should show edit icon when it has been reached, but not active', () => {
    render(
      <TestWrapper>
        <PreviewSection title="My title" order={2} hasBeenReached>
          Some text
        </PreviewSection>
      </TestWrapper>
    );

    expect(screen.queryByTestId('edit-section-2')).toBeInTheDocument();
  });

  it('should not show edit icon when active', () => {
    render(
      <TestWrapper>
        <PreviewSection title="My title" order={2} hasBeenReached active>
          Some text
        </PreviewSection>
      </TestWrapper>
    );

    expect(screen.queryByTestId('edit-section-2')).not.toBeInTheDocument();
  });

  it('should trigger edit callback', () => {
    const cb = vi.fn();

    render(
      <TestWrapper>
        <PreviewSection
          onEditClick={cb}
          title="My title"
          order={2}
          hasBeenReached
        >
          Some text
        </PreviewSection>
      </TestWrapper>
    );

    fireEvent.click(screen.getByTestId('edit-section-2'));
    expect(cb).toHaveBeenCalled();
  });
});

describe('TruncatedPreviewText', () => {
  let originalResizeObserver: typeof ResizeObserver | undefined;

  beforeAll(() => {
    originalResizeObserver = global.ResizeObserver;
    // Mock ResizeObserver
    class ResizeObserverMock implements ResizeObserver {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
    global.ResizeObserver = ResizeObserverMock;
  });

  afterAll(() => {
    if (originalResizeObserver !== undefined) {
      global.ResizeObserver = originalResizeObserver;
    } else {
      Reflect.deleteProperty(global, 'ResizeObserver');
    }
  });

  it('should render the provided text', () => {
    render(
      <TestWrapper>
        <TruncatedPreviewText text="Short value" dataTestId="test" />
      </TestWrapper>
    );

    expect(
      screen.getByTestId('test-preview-content')
    ).toHaveTextContent('Short value');
  });

  it('should use default test id when dataTestId is not provided', () => {
    render(
      <TestWrapper>
        <TruncatedPreviewText text="Some text" />
      </TestWrapper>
    );

    expect(screen.getByTestId('preview-content')).toBeInTheDocument();
    expect(screen.getByTestId('truncated-wrapper')).toBeInTheDocument();
  });

  it('should use custom test ids when dataTestId is provided', () => {
    render(
      <TestWrapper>
        <TruncatedPreviewText text="Some text" dataTestId="engine" />
      </TestWrapper>
    );

    expect(screen.getByTestId('engine-preview-content')).toBeInTheDocument();
    expect(
      screen.getByTestId('engine-truncated-wrapper')
    ).toBeInTheDocument();
  });

  it('should toggle expanded state when clicking the toggle', () => {
    // Mock scrollHeight to simulate overflow
    const originalScrollHeight = Object.getOwnPropertyDescriptor(
      HTMLElement.prototype,
      'scrollHeight'
    );
    Object.defineProperty(HTMLElement.prototype, 'scrollHeight', {
      configurable: true,
      get() {
        return 100;
      },
    });
    const originalClientHeight = Object.getOwnPropertyDescriptor(
      HTMLElement.prototype,
      'clientHeight'
    );
    Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
      configurable: true,
      get() {
        return 20;
      },
    });

    render(
      <TestWrapper>
        <TruncatedPreviewText
          text="A very long text that should overflow"
          dataTestId="test"
        />
      </TestWrapper>
    );

    const toggle = screen.getByTestId('test-truncated-toggle');

    // Initially collapsed
    expect(toggle).toHaveTextContent('Show more');
    expect(toggle).toHaveAttribute('aria-expanded', 'false');

    // Click to expand
    fireEvent.click(toggle);
    expect(toggle).toHaveTextContent('Show less');
    expect(toggle).toHaveAttribute('aria-expanded', 'true');

    // Click to collapse again
    fireEvent.click(toggle);
    expect(toggle).toHaveTextContent('Show more');
    expect(toggle).toHaveAttribute('aria-expanded', 'false');

    // Restore original property descriptors
    if (originalScrollHeight) {
      Object.defineProperty(
        HTMLElement.prototype,
        'scrollHeight',
        originalScrollHeight
      );
    }
    if (originalClientHeight) {
      Object.defineProperty(
        HTMLElement.prototype,
        'clientHeight',
        originalClientHeight
      );
    }
  });

  it('should not render toggle when text does not overflow', () => {
    // Mock scrollHeight equal to clientHeight to simulate no overflow
    const originalScrollHeight = Object.getOwnPropertyDescriptor(
      HTMLElement.prototype,
      'scrollHeight'
    );
    Object.defineProperty(HTMLElement.prototype, 'scrollHeight', {
      configurable: true,
      get() {
        return 20;
      },
    });
    const originalClientHeight = Object.getOwnPropertyDescriptor(
      HTMLElement.prototype,
      'clientHeight'
    );
    Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
      configurable: true,
      get() {
        return 20;
      },
    });

    render(
      <TestWrapper>
        <TruncatedPreviewText text="Short" dataTestId="test" />
      </TestWrapper>
    );

    expect(
      screen.queryByTestId('test-truncated-toggle')
    ).not.toBeInTheDocument();

    // Restore original property descriptors
    if (originalScrollHeight) {
      Object.defineProperty(
        HTMLElement.prototype,
        'scrollHeight',
        originalScrollHeight
      );
    }
    if (originalClientHeight) {
      Object.defineProperty(
        HTMLElement.prototype,
        'clientHeight',
        originalClientHeight
      );
    }
  });

  it('should reset expanded state when text prop changes', () => {
    // Mock scrollHeight > clientHeight
    const originalScrollHeight = Object.getOwnPropertyDescriptor(
      HTMLElement.prototype,
      'scrollHeight'
    );
    Object.defineProperty(HTMLElement.prototype, 'scrollHeight', {
      configurable: true,
      get() {
        return 100;
      },
    });
    const originalClientHeight = Object.getOwnPropertyDescriptor(
      HTMLElement.prototype,
      'clientHeight'
    );
    Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
      configurable: true,
      get() {
        return 20;
      },
    });

    const { rerender } = render(
      <TestWrapper>
        <TruncatedPreviewText
          text="Initial long text"
          dataTestId="reset-test"
        />
      </TestWrapper>
    );

    const toggle = screen.getByTestId('reset-test-truncated-toggle');

    // Expand
    fireEvent.click(toggle);
    expect(toggle).toHaveAttribute('aria-expanded', 'true');

    // Re-render with new text
    rerender(
      <TestWrapper>
        <TruncatedPreviewText text="New long text" dataTestId="reset-test" />
      </TestWrapper>
    );

    // Should be collapsed again
    expect(screen.getByTestId('reset-test-truncated-toggle')).toHaveAttribute(
      'aria-expanded',
      'false'
    );

    // Restore mocks
    if (originalScrollHeight) {
      Object.defineProperty(
        HTMLElement.prototype,
        'scrollHeight',
        originalScrollHeight
      );
    }
    if (originalClientHeight) {
      Object.defineProperty(
        HTMLElement.prototype,
        'clientHeight',
        originalClientHeight
      );
    }
  });
});
