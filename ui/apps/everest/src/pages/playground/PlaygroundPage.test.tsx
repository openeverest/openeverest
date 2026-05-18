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
import { PlaygroundPage } from './PlaygroundPage';
import { TestWrapper } from 'utils/test';

describe('PlaygroundPage', () => {
  it('should render the page with title', () => {
    render(
      <TestWrapper>
        <PlaygroundPage />
      </TestWrapper>
    );

    expect(screen.getByTestId('playground-page')).toBeInTheDocument();
    expect(screen.getByText('YAML Playground')).toBeInTheDocument();
  });

  it('should show the PoC badge', () => {
    render(
      <TestWrapper>
        <PlaygroundPage />
      </TestWrapper>
    );

    expect(screen.getByTestId('poc-badge')).toBeInTheDocument();
    expect(screen.getByText('PoC')).toBeInTheDocument();
  });

  it('should render the editor and preview layout', () => {
    render(
      <TestWrapper>
        <PlaygroundPage />
      </TestWrapper>
    );

    expect(screen.getByTestId('playground-layout')).toBeInTheDocument();
    expect(screen.getByTestId('yaml-editor')).toBeInTheDocument();
    expect(screen.getByTestId('preview-panel')).toBeInTheDocument();
  });

  it('should show valid YAML status for default content', () => {
    render(
      <TestWrapper>
        <PlaygroundPage />
      </TestWrapper>
    );

    expect(screen.getByTestId('status-valid')).toBeInTheDocument();
    expect(screen.getByText('Valid YAML')).toBeInTheDocument();
  });

  it('should render default YAML content in the editor', () => {
    render(
      <TestWrapper>
        <PlaygroundPage />
      </TestWrapper>
    );

    const editor = screen.getByTestId('yaml-editor');
    const cmContent = editor.querySelector('.cm-content');
    expect(cmContent?.textContent).toContain('DatabaseCluster');
  });

  it('should render parsed preview for default content', () => {
    render(
      <TestWrapper>
        <PlaygroundPage />
      </TestWrapper>
    );

    expect(screen.getByTestId('preview-output')).toBeInTheDocument();
    const output = screen.getByTestId('preview-output');
    expect(output.textContent).toContain('"apiVersion"');
  });

  it('should have accessible heading structure', () => {
    render(
      <TestWrapper>
        <PlaygroundPage />
      </TestWrapper>
    );

    const heading = screen.getByRole('heading', { level: 1 });
    expect(heading).toHaveTextContent('YAML Playground');
  });
});
