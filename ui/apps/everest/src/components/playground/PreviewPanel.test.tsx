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
import { PreviewPanel } from './PreviewPanel';
import { TestWrapper } from 'utils/test';

describe('PreviewPanel', () => {
  it('should show empty state when no content is provided', () => {
    render(
      <TestWrapper>
        <PreviewPanel parsed={null} error={null} />
      </TestWrapper>
    );

    expect(screen.getByTestId('preview-empty')).toBeInTheDocument();
    expect(screen.getByText(/enter yaml/i)).toBeInTheDocument();
  });

  it('should show validation error when present', () => {
    render(
      <TestWrapper>
        <PreviewPanel parsed={null} error="Bad indentation at line 3" />
      </TestWrapper>
    );

    expect(screen.getByTestId('preview-error')).toBeInTheDocument();
    expect(
      screen.getByText('Bad indentation at line 3')
    ).toBeInTheDocument();
    expect(screen.queryByTestId('preview-output')).not.toBeInTheDocument();
  });

  it('should show parsed JSON when YAML is valid', () => {
    const parsed = { name: 'test', replicas: 3 };
    render(
      <TestWrapper>
        <PreviewPanel parsed={parsed} error={null} />
      </TestWrapper>
    );

    expect(screen.getByTestId('preview-output')).toBeInTheDocument();
    const output = screen.getByTestId('preview-output');
    expect(output.textContent).toContain('"name": "test"');
    expect(output.textContent).toContain('"replicas": 3');
  });

  it('should show parsed output header', () => {
    render(
      <TestWrapper>
        <PreviewPanel parsed={{ key: 'value' }} error={null} />
      </TestWrapper>
    );

    expect(
      screen.getByText('Parsed Output (JSON)')
    ).toBeInTheDocument();
  });

  it('should handle complex nested structures', () => {
    const parsed = {
      metadata: { name: 'db', labels: { app: 'everest' } },
      spec: { items: [1, 2, 3] },
    };

    render(
      <TestWrapper>
        <PreviewPanel parsed={parsed} error={null} />
      </TestWrapper>
    );

    const output = screen.getByTestId('preview-output');
    expect(output.textContent).toContain('"app": "everest"');
  });

  it('should prioritize error display over parsed content', () => {
    // If both error and parsed are provided, error should win
    render(
      <TestWrapper>
        <PreviewPanel
          parsed={{ stale: 'data' }}
          error="New parse error"
        />
      </TestWrapper>
    );

    expect(screen.getByTestId('preview-error')).toBeInTheDocument();
    expect(screen.queryByTestId('preview-output')).not.toBeInTheDocument();
  });
});
