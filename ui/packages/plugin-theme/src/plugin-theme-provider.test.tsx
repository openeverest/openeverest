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

import { act, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useTheme } from '@mui/material';
import { PluginThemeProvider } from './plugin-theme-provider';

// Exercises the host<->plugin theming bridge: a plugin subtree rendered under a
// host that publishes `--everest-*` tokens should inherit the host palette,
// typography and shape, and re-render when the host toggles dark mode. jsdom
// doesn't resolve custom properties through getComputedStyle, so we back it onto
// the element's inline style — which is where the tests write the tokens.
function setToken(name: string, value: string): void {
  document.documentElement.style.setProperty(name, value);
}

function ThemeProbe() {
  const theme = useTheme();
  return (
    <div>
      <span data-testid="mode">{theme.palette.mode}</span>
      <span data-testid="primary">{theme.palette.primary.main}</span>
      <span data-testid="bg">{theme.palette.background?.default ?? ''}</span>
      <span data-testid="radius">{String(theme.shape.borderRadius)}</span>
      <span data-testid="h1-family">
        {String(theme.typography.h1.fontFamily ?? '')}
      </span>
      <span data-testid="btn-transform">
        {String(theme.typography.button.textTransform ?? '')}
      </span>
      <span data-testid="btn-spacing">
        {String(theme.typography.button.letterSpacing ?? '')}
      </span>
      <span data-testid="caption-transform">
        {String(theme.typography.caption.textTransform ?? '')}
      </span>
    </div>
  );
}

function renderUnderHost() {
  return render(
    <PluginThemeProvider cacheKey="bridge-test">
      <ThemeProbe />
    </PluginThemeProvider>
  );
}

describe('PluginThemeProvider host token bridge', () => {
  beforeEach(() => {
    vi.spyOn(window, 'getComputedStyle').mockImplementation(
      (el: Element) =>
        ({
          getPropertyValue: (prop: string) =>
            (el as HTMLElement).style.getPropertyValue(prop),
        }) as CSSStyleDeclaration
    );
    document.documentElement.setAttribute('data-everest-color-scheme', 'light');
    setToken('--everest-color-primary-main', 'rgb(14, 95, 181)');
    setToken('--everest-color-background-default', 'rgb(255, 255, 255)');
    setToken('--everest-radius', '4');
  });

  afterEach(() => {
    vi.restoreAllMocks();
    document.documentElement.removeAttribute('data-everest-color-scheme');
    document.documentElement.removeAttribute('style');
  });

  it('inherits the host palette and shape from --everest-* tokens', () => {
    renderUnderHost();
    expect(screen.getByTestId('mode')).toHaveTextContent('light');
    expect(screen.getByTestId('primary')).toHaveTextContent('rgb(14, 95, 181)');
    expect(screen.getByTestId('bg')).toHaveTextContent('rgb(255, 255, 255)');
    expect(screen.getByTestId('radius')).toHaveTextContent('4');
  });

  it('carries per-property typography including textTransform and letterSpacing', () => {
    // The whole reason for per-property tokens: the `font` shorthand can't carry
    // these, so buttons would stay UPPERCASE without them.
    setToken('--everest-font-h1-family', "'Poppins', sans-serif");
    setToken('--everest-font-button-transform', 'none');
    setToken('--everest-font-button-letter-spacing', '0.025em');
    renderUnderHost();
    expect(screen.getByTestId('h1-family')).toHaveTextContent(
      "'Poppins', sans-serif"
    );
    expect(screen.getByTestId('btn-transform')).toHaveTextContent('none');
    expect(screen.getByTestId('btn-spacing')).toHaveTextContent('0.025em');
  });

  it('ignores a non-value transform token instead of applying it verbatim', () => {
    setToken('--everest-font-caption-transform', 'inherit');
    renderUnderHost();
    expect(screen.getByTestId('caption-transform')).not.toHaveTextContent(
      'inherit'
    );
  });

  it('re-renders with dark values when the host toggles the color scheme', async () => {
    renderUnderHost();
    expect(screen.getByTestId('primary')).toHaveTextContent('rgb(14, 95, 181)');

    await act(async () => {
      setToken('--everest-color-primary-main', 'rgb(98, 174, 255)');
      document.documentElement.setAttribute(
        'data-everest-color-scheme',
        'dark'
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId('mode')).toHaveTextContent('dark');
      expect(screen.getByTestId('primary')).toHaveTextContent(
        'rgb(98, 174, 255)'
      );
    });
  });
});
