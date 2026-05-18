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

import { screen, render } from '@testing-library/react';
import { CodeMirrorYamlEditor } from './CodeMirrorYamlEditor';
import { TestWrapper } from 'utils/test';

describe('CodeMirrorYamlEditor', () => {
  it('should render the editor container with correct test id', () => {
    render(
      <TestWrapper>
        <CodeMirrorYamlEditor />
      </TestWrapper>
    );

    expect(screen.getByTestId('yaml-editor')).toBeInTheDocument();
  });

  it('should have correct accessibility attributes', () => {
    render(
      <TestWrapper>
        <CodeMirrorYamlEditor />
      </TestWrapper>
    );

    const editor = screen.getByTestId('yaml-editor');
    expect(editor).toHaveAttribute('role', 'textbox');
    expect(editor).toHaveAttribute('aria-label', 'YAML editor');
    expect(editor).toHaveAttribute('aria-multiline', 'true');
  });

  it('should render with initial content', () => {
    render(
      <TestWrapper>
        <CodeMirrorYamlEditor initialContent="name: test" />
      </TestWrapper>
    );

    const editor = screen.getByTestId('yaml-editor');
    // CodeMirror renders content inside .cm-content
    const cmContent = editor.querySelector('.cm-content');
    expect(cmContent).toBeInTheDocument();
    expect(cmContent?.textContent).toContain('name: test');
  });

  it('should create a CodeMirror editor instance inside the container', () => {
    render(
      <TestWrapper>
        <CodeMirrorYamlEditor initialContent="key: value" />
      </TestWrapper>
    );

    const editor = screen.getByTestId('yaml-editor');
    // Verify CodeMirror's root element is present
    expect(editor.querySelector('.cm-editor')).toBeInTheDocument();
  });
});
