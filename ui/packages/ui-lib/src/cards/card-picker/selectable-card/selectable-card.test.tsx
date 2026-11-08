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

import { fireEvent, render, screen } from '@testing-library/react';
import { ThemeProvider, createTheme } from '@mui/material';
import { everestThemeOptions } from '@percona/design';
import { SelectableCard } from './selectable-card';

const theme = createTheme(everestThemeOptions('light'));
const item = { id: 'pmm-prod', title: 'pmm-prod', subtitle: 'PMM' };

const renderCard = (props: { selected?: boolean; disabled?: boolean }) => {
  const onSelect = vi.fn();
  render(
    <ThemeProvider theme={theme}>
      <SelectableCard
        item={item}
        selected={props.selected ?? false}
        disabled={props.disabled}
        onSelect={onSelect}
      />
    </ThemeProvider>
  );
  return onSelect;
};

describe('SelectableCard', () => {
  it('is a toggle button reporting its selection', () => {
    const onSelect = renderCard({ selected: true });
    const card = screen.getByRole('button', { name: /pmm-prod/ });

    expect(card).toHaveAttribute('aria-pressed', 'true');
    fireEvent.click(card);
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it('cannot be picked while disabled', () => {
    const onSelect = renderCard({ disabled: true });

    expect(screen.queryByRole('button')).not.toBeInTheDocument();
    fireEvent.click(screen.getByText('pmm-prod'));
    expect(onSelect).not.toHaveBeenCalled();
  });
});
