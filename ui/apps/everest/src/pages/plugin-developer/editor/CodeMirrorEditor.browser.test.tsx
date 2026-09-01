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

import { useState } from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor, cleanup } from '@testing-library/react';
import { userEvent } from 'vitest/browser';
import { CodeMirrorEditor } from './CodeMirrorEditor';
import type { Diagnostic } from './types';

const NONCE = 'test-nonce-123';

beforeEach(() => {
  const meta = document.createElement('meta');
  meta.setAttribute('name', 'csp-nonce');
  meta.setAttribute('content', NONCE);
  document.head.appendChild(meta);
});

afterEach(() => {
  cleanup();
  document.querySelector("meta[name='csp-nonce']")?.remove();
});

describe('CodeMirrorEditor (in-tree CM6)', () => {
  it('mounts a CodeMirror editor showing the initial value', async () => {
    const { container } = render(
      <CodeMirrorEditor
        value="hello: world"
        onChange={() => {}}
        diagnostics={[]}
        theme="light"
      />
    );
    await waitFor(() =>
      expect(container.querySelector('.cm-editor')).not.toBeNull()
    );
    expect(container.querySelector('.cm-content')!.textContent).toContain(
      'hello: world'
    );
  });

  it('stamps CM-injected styles with the app CSP nonce', async () => {
    render(
      <CodeMirrorEditor
        value="a: 1"
        onChange={() => {}}
        diagnostics={[]}
        theme="light"
      />
    );
    await waitFor(() =>
      expect(document.querySelector(`style[nonce="${NONCE}"]`)).not.toBeNull()
    );
  });

  it('calls onChange (debounced) when the user types', async () => {
    const onChange = vi.fn();
    const { container } = render(
      <CodeMirrorEditor
        value=""
        onChange={onChange}
        diagnostics={[]}
        theme="light"
      />
    );
    await waitFor(() =>
      expect(container.querySelector('.cm-content')).not.toBeNull()
    );
    const content = container.querySelector('.cm-content') as HTMLElement;
    await userEvent.click(content);
    await userEvent.type(content, 'x: 1');
    await waitFor(() =>
      expect(onChange).toHaveBeenCalledWith(expect.stringContaining('x: 1'))
    );
  });

  it('reflects an external value change (Format YAML) into the editor', async () => {
    const { container, rerender } = render(
      <CodeMirrorEditor
        value="a: 1"
        onChange={() => {}}
        diagnostics={[]}
        theme="light"
      />
    );
    await waitFor(() =>
      expect(container.querySelector('.cm-content')!.textContent).toContain(
        'a: 1'
      )
    );
    rerender(
      <CodeMirrorEditor
        value="b: 2"
        onChange={() => {}}
        diagnostics={[]}
        theme="light"
      />
    );
    await waitFor(() =>
      expect(container.querySelector('.cm-content')!.textContent).toContain(
        'b: 2'
      )
    );
    expect(container.querySelector('.cm-content')!.textContent).not.toContain(
      'a: 1'
    );
  });

  it('renders a lint marker for an error diagnostic', async () => {
    const diagnostics: Diagnostic[] = [
      { line: 1, column: 1, message: 'bad', severity: 'error' },
    ];
    const { container } = render(
      <CodeMirrorEditor
        value="oops: [unterminated"
        onChange={() => {}}
        diagnostics={diagnostics}
        theme="light"
      />
    );
    await waitFor(() =>
      expect(container.querySelector('.cm-lint-marker-error')).not.toBeNull()
    );
  });

  it('does not drop keystrokes when onChange is threaded back in as value (debounce/value-sync race)', async () => {
    // Controlled harness: mirrors how the real parent uses this component -
    // onChange updates React state, which flows back in as the `value` prop.
    // This is the loop where a keystroke landing between the debounced
    // onChange firing and the value-sync effect running can otherwise get
    // clobbered by a stale, older `value`.
    const onChange = vi.fn();
    const Harness = () => {
      const [value, setValue] = useState('');
      const handleChange = (next: string) => {
        onChange(next);
        setValue(next);
      };
      return (
        <CodeMirrorEditor
          value={value}
          onChange={handleChange}
          diagnostics={[]}
          theme="light"
        />
      );
    };

    const { container } = render(<Harness />);
    await waitFor(() =>
      expect(container.querySelector('.cm-content')).not.toBeNull()
    );
    const content = container.querySelector('.cm-content') as HTMLElement;
    await userEvent.click(content);

    const typed = 'hello: world-1234';
    await userEvent.type(content, typed);

    await waitFor(() => {
      expect(container.querySelector('.cm-content')!.textContent).toBe(typed);
    });
    await waitFor(() => {
      expect(onChange).toHaveBeenCalled();
      const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1];
      expect(lastCall[0]).toBe(typed);
    });
  });

  it('fires onChange on undo, keeping the controlled value in sync (Ctrl+Z)', async () => {
    // Same controlled harness as above: onChange feeds back into `value`. An
    // undo that doesn't emit onChange would leave the doc reverted but the
    // parent's `value` state stale - a silent desync / data-loss path.
    const onChange = vi.fn();
    const Harness = () => {
      const [value, setValue] = useState('');
      const handleChange = (next: string) => {
        onChange(next);
        setValue(next);
      };
      return (
        <CodeMirrorEditor
          value={value}
          onChange={handleChange}
          diagnostics={[]}
          theme="light"
        />
      );
    };

    const { container } = render(<Harness />);
    await waitFor(() =>
      expect(container.querySelector('.cm-content')).not.toBeNull()
    );
    const content = container.querySelector('.cm-content') as HTMLElement;
    await userEvent.click(content);

    const typed = 'x: 1';
    await userEvent.type(content, typed);
    await waitFor(() => {
      expect(container.querySelector('.cm-content')!.textContent).toBe(typed);
    });
    // Wait for the typing debounce to flush before capturing the baseline,
    // so the undo's own onChange call is unambiguous.
    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith(typed);
    });
    const callsBeforeUndo = onChange.mock.calls.length;

    await userEvent.keyboard('{Control>}z{/Control}');

    await waitFor(() => {
      expect(onChange.mock.calls.length).toBeGreaterThan(callsBeforeUndo);
    });
    await waitFor(() => {
      const editorText = container.querySelector('.cm-content')!.textContent;
      const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1];
      // The reverted doc text and the harness's controlled `value` (fed by
      // onChange) must agree - that's the desync this test guards against.
      expect(lastCall[0]).toBe(editorText);
    });
  });
});
