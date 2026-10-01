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

import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TestWrapper } from 'utils/test';
import { OutputPanel } from './output-panel';
import { Messages } from './output-panel.messages';

const SAMPLE = { engine: { type: 'pxc' } };

describe('OutputPanel', () => {
  it('renders the empty-state message and no Copy button when payload is null', () => {
    render(
      <TestWrapper>
        <OutputPanel payload={null} />
      </TestWrapper>
    );

    expect(screen.getByText(Messages.emptyState)).toBeInTheDocument();
    expect(screen.queryByText(Messages.copy)).not.toBeInTheDocument();
    expect(screen.queryByTestId('output-json')).not.toBeInTheDocument();
  });

  it('renders the pretty-printed JSON payload', () => {
    render(
      <TestWrapper>
        <OutputPanel payload={SAMPLE} />
      </TestWrapper>
    );

    const output = screen.getByTestId('output-json');
    expect(output.textContent).toBe(JSON.stringify(SAMPLE, null, 2));
  });

  it('copies the pretty JSON to the clipboard when Copy is clicked', () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    render(
      <TestWrapper>
        <OutputPanel payload={SAMPLE} />
      </TestWrapper>
    );

    fireEvent.click(screen.getByText(Messages.copy));

    expect(writeText).toHaveBeenCalledWith(JSON.stringify(SAMPLE, null, 2));
  });
});
