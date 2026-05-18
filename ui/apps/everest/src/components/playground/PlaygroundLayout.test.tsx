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
import { PlaygroundLayout } from './PlaygroundLayout';
import { TestWrapper } from 'utils/test';

describe('PlaygroundLayout', () => {
  it('should render both editor and preview panes', () => {
    render(
      <TestWrapper>
        <PlaygroundLayout
          editor={<div data-testid="test-editor">Editor</div>}
          preview={<div data-testid="test-preview">Preview</div>}
        />
      </TestWrapper>
    );

    expect(screen.getByTestId('playground-layout')).toBeInTheDocument();
    expect(screen.getByTestId('test-editor')).toBeInTheDocument();
    expect(screen.getByTestId('test-preview')).toBeInTheDocument();
  });

  it('should have accessible pane labels', () => {
    render(
      <TestWrapper>
        <PlaygroundLayout
          editor={<span>E</span>}
          preview={<span>P</span>}
        />
      </TestWrapper>
    );

    expect(screen.getByLabelText('Editor pane')).toBeInTheDocument();
    expect(screen.getByLabelText('Preview pane')).toBeInTheDocument();
  });

  it('should render a vertical divider between panes', () => {
    const { container } = render(
      <TestWrapper>
        <PlaygroundLayout
          editor={<span>E</span>}
          preview={<span>P</span>}
        />
      </TestWrapper>
    );

    // MUI Divider renders an <hr> element
    const divider = container.querySelector('hr');
    expect(divider).toBeInTheDocument();
  });
});
