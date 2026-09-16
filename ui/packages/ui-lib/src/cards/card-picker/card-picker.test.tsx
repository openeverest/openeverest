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

import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ReactNode } from 'react';
import { ThemeProvider, createTheme } from '@mui/material';
import { everestThemeOptions } from '@percona/design';

// Disable transitions so the Dialog opens/closes instantly — keeps the modal
// test deterministic and fast under parallel test load.
const theme = createTheme(everestThemeOptions('light'), {
  transitions: {
    create: () => 'none',
    duration: {
      shortest: 0,
      shorter: 0,
      short: 0,
      standard: 0,
      complex: 0,
      enteringScreen: 0,
      leavingScreen: 0,
    },
  },
});
const TestWrapper = ({ children }: { children: ReactNode }) => (
  <ThemeProvider theme={theme}>{children}</ThemeProvider>
);

// Rendering the MUI card grid + modal is render-heavy; give these integration
// tests headroom so they don't flake under parallel test load.
vi.setConfig({ testTimeout: 15000 });

// Control the responsive column count so inline-vs-overflow is deterministic
// without a real ResizeObserver / layout.
const { columnsRef } = vi.hoisted(() => ({ columnsRef: { value: 12 } }));
vi.mock('./use-grid-column-count', () => ({
  useGridColumnCount: () => columnsRef.value,
}));

import { CardPicker } from './card-picker';
import { CardPickerItem } from './card-picker.types';

const items: CardPickerItem[] = [
  { id: 'p1', title: 'p1', subtitle: 'replicaSet', detail: '3 nodes' },
  { id: 'p2', title: 'p2', subtitle: 'sharded', detail: '5 nodes' },
  {
    id: 'p3',
    title: 'p3',
    subtitle: 'replicaSet',
    detail: '7 nodes',
    searchText: 'p3 storageclass local-path',
  },
  { id: 'p4', title: 'p4', subtitle: 'sharded', detail: '3 nodes' },
  { id: 'p5', title: 'p5', subtitle: 'replicaSet', detail: '5 nodes' },
];
const lead: CardPickerItem = { id: '', title: 'Scratch', subtitle: 'manual' };

const setup = (overrides?: {
  selectedId?: string;
  onSelect?: (id: string) => void;
}) => {
  const onSelect = overrides?.onSelect ?? vi.fn();
  render(
    <TestWrapper>
      <CardPicker
        items={items}
        leadCard={lead}
        selectedId={overrides?.selectedId ?? ''}
        onSelect={onSelect}
      />
    </TestWrapper>
  );
  return { onSelect };
};

describe('CardPicker', () => {
  beforeEach(() => {
    columnsRef.value = 12;
  });

  it('renders every item inline with no Browse tile when they all fit', () => {
    setup();

    expect(screen.getByRole('button', { name: /Scratch/ })).toBeInTheDocument();
    ['p1', 'p2', 'p3', 'p4', 'p5'].forEach((id) =>
      expect(
        screen.getByRole('button', { name: new RegExp(id) })
      ).toBeInTheDocument()
    );
    expect(screen.queryByText(/Browse all/i)).not.toBeInTheDocument();
  });

  it('shows a Browse tile and only the fitting items when they overflow', () => {
    columnsRef.value = 3; // scratch + 1 preset + Browse
    setup();

    expect(screen.getByRole('button', { name: /p1/ })).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /p2/ })
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /Browse all 5/ })
    ).toBeInTheDocument();
  });

  it('keeps the selected item visible inline even past the fitting slice', () => {
    columnsRef.value = 3;
    setup({ selectedId: 'p5' });

    expect(screen.getByRole('button', { name: /p5/ })).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /p1/ })
    ).not.toBeInTheDocument();
  });

  it('marks the selected card with aria-pressed', () => {
    setup({ selectedId: 'p2' });

    expect(screen.getByRole('button', { name: /p2/ })).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    expect(screen.getByRole('button', { name: /Scratch/ })).toHaveAttribute(
      'aria-pressed',
      'false'
    );
  });

  it('opens the modal, searches across searchText and selects an item', async () => {
    columnsRef.value = 3;
    const { onSelect } = setup();

    fireEvent.click(screen.getByRole('button', { name: /Browse all 5/ }));
    const dialog = await screen.findByRole('dialog');
    ['p1', 'p2', 'p3', 'p4', 'p5'].forEach((id) =>
      expect(
        within(dialog).getByRole('button', { name: new RegExp(id) })
      ).toBeInTheDocument()
    );

    const search = within(dialog).getByRole('textbox', {
      name: 'Search options',
    });

    // A spec-only token matches p3 through searchText, though it isn't shown.
    fireEvent.change(search, { target: { value: 'storageclass' } });
    expect(
      within(dialog).getByRole('button', { name: /p3/ })
    ).toBeInTheDocument();
    expect(
      within(dialog).queryByRole('button', { name: /p1/ })
    ).not.toBeInTheDocument();

    fireEvent.change(search, { target: { value: 'zzz' } });
    expect(within(dialog).getByText(/No matches for/i)).toBeInTheDocument();

    fireEvent.change(search, { target: { value: 'p4' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /p4/ }));

    expect(onSelect).toHaveBeenCalledWith('p4');
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    );
  });

  it('renders a disabled status card when statusCard is provided', () => {
    render(
      <TestWrapper>
        <CardPicker
          items={[]}
          leadCard={lead}
          selectedId=""
          onSelect={vi.fn()}
          statusCard={{ title: 'Presets did not load', subtitle: 'Retry.' }}
        />
      </TestWrapper>
    );

    expect(screen.getByText('Presets did not load')).toBeInTheDocument();
    expect(screen.getByText('Retry.')).toBeInTheDocument();
    // The notice is informational, not a selectable/clickable card.
    expect(
      screen.queryByRole('button', { name: /Presets did not load/ })
    ).not.toBeInTheDocument();
  });
});
