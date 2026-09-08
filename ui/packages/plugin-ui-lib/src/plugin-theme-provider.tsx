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

import { useEffect, useMemo, useState, type ReactNode } from 'react';
import createCache from '@emotion/cache';
import { CacheProvider } from '@emotion/react';
import {
  createTheme,
  ThemeProvider,
  type PaletteMode,
  type PaletteOptions,
  type SimplePaletteColorOptions,
  type ThemeOptions,
} from '@mui/material';

// The host publishes its active color scheme here (see @percona/design
// ThemeContextProvider). Plugins can't read the host React context because
// they bundle their own MUI, so they observe this attribute instead.
const HOST_COLOR_SCHEME_ATTR = 'data-everest-color-scheme';

// Reads a host-owned `--mui-*` CSS variable's *computed* value. We resolve to
// concrete colors rather than passing `var(...)` into the theme, because MUI's
// color functions (alpha/darken/lighten) can't parse `var()` strings (error #9).
function readVar(name: string): string {
  if (typeof document === 'undefined') {
    return '';
  }
  return getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim();
}

function colorFromHost(name: string): SimplePaletteColorOptions | undefined {
  const main = readVar(`--mui-palette-${name}-main`);
  if (!main) {
    return undefined;
  }
  const color: SimplePaletteColorOptions = { main };
  const dark = readVar(`--mui-palette-${name}-dark`);
  const light = readVar(`--mui-palette-${name}-light`);
  if (dark) color.dark = dark;
  if (light) color.light = light;
  // contrastText is intentionally NOT read from the host: some host tokens set a
  // low-contrast value; letting MUI derive it from `main` keeps filled chips/
  // buttons readable.
  return color;
}

// Builds a palette of concrete colors read from the host CSS variables. Missing
// tokens fall back to MUI defaults. Re-run whenever the host color scheme flips.
function hostPalette(mode: PaletteMode): PaletteOptions {
  const palette: PaletteOptions = { mode };

  const primary = colorFromHost('primary');
  const secondary = colorFromHost('secondary');
  const error = colorFromHost('error');
  const warning = colorFromHost('warning');
  const info = colorFromHost('info');
  const success = colorFromHost('success');
  if (primary) palette.primary = primary;
  if (secondary) palette.secondary = secondary;
  if (error) palette.error = error;
  if (warning) palette.warning = warning;
  if (info) palette.info = info;
  if (success) palette.success = success;

  const textPrimary = readVar('--mui-palette-text-primary');
  if (textPrimary) {
    const secondaryText = readVar('--mui-palette-text-secondary');
    const disabledText = readVar('--mui-palette-text-disabled');
    palette.text = {
      primary: textPrimary,
      ...(secondaryText && { secondary: secondaryText }),
      ...(disabledText && { disabled: disabledText }),
    };
  }

  const bgDefault = readVar('--mui-palette-background-default');
  if (bgDefault) {
    const paper = readVar('--mui-palette-background-paper');
    palette.background = {
      default: bgDefault,
      ...(paper && { paper }),
    };
  }

  const divider = readVar('--mui-palette-divider');
  if (divider) {
    palette.divider = divider;
  }

  return palette;
}

// The typography variants MUI emits as `--mui-font-*`.
const TYPOGRAPHY_VARIANTS = [
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'subtitle1',
  'subtitle2',
  'body1',
  'body2',
  'button',
  'caption',
  'overline',
] as const;

// MUI emits each variant as a CSS `font` shorthand:
//   "<weight> <size>[/<lineHeight>] [<family>]"
// e.g. "600 1.5rem/22.5px 'Poppins',sans-serif".
//
// Note the shorthand cannot express `textTransform` or `letterSpacing`, so those
// host values do not cross this bridge — see the note in §7.4.
function fontFromHost(variant: string): Record<string, unknown> | undefined {
  const raw = readVar(`--mui-font-${variant}`);
  if (!raw) {
    return undefined;
  }
  // Requiring a numeric weight also rejects non-font values such as
  // `--mui-font-inherit: inherit inherit/inherit inherit`.
  const m = raw.match(/^(\d+)\s+([^\s/]+)(?:\/([^\s]+))?(?:\s+(.+))?$/);
  if (!m) {
    return undefined;
  }
  const [, weight, size, lineHeight, family] = m;
  const style: Record<string, unknown> = {
    fontWeight: Number(weight),
    fontSize: size,
  };
  if (lineHeight) {
    style.lineHeight = lineHeight;
  }
  if (family) {
    style.fontFamily = family;
  }
  return style;
}

// Builds the typography scale from host tokens. Variants the host doesn't
// publish simply fall back to the plugin's own MUI defaults.
function hostTypography(): ThemeOptions['typography'] {
  const typography: Record<string, unknown> = {};
  for (const variant of TYPOGRAPHY_VARIANTS) {
    const style = fontFromHost(variant);
    if (style) {
      typography[variant] = style;
    }
  }
  // Give unlisted variants the host's body face rather than MUI's default.
  const bodyFamily = (typography.body1 as Record<string, unknown> | undefined)
    ?.fontFamily;
  if (bodyFamily) {
    typography.fontFamily = bodyFamily;
  }
  return Object.keys(typography).length
    ? (typography as ThemeOptions['typography'])
    : undefined;
}

function hostShape(): ThemeOptions['shape'] {
  const radius = parseFloat(readVar('--mui-shape-borderRadius'));
  return Number.isFinite(radius) ? { borderRadius: radius } : undefined;
}

function readHostColorScheme(): PaletteMode {
  if (typeof document === 'undefined') {
    return 'light';
  }
  const value = document.documentElement.getAttribute(HOST_COLOR_SCHEME_ATTR);
  return value === 'dark' ? 'dark' : 'light';
}

// Tracks the host color scheme so plugin components re-render on dark-mode toggle.
export function useHostColorMode(): PaletteMode {
  const [mode, setMode] = useState<PaletteMode>(readHostColorScheme);

  useEffect(() => {
    const target = document.documentElement;
    const observer = new MutationObserver(() => setMode(readHostColorScheme()));
    observer.observe(target, {
      attributes: true,
      attributeFilter: [HOST_COLOR_SCHEME_ATTR],
    });
    setMode(readHostColorScheme());
    return () => observer.disconnect();
  }, []);

  return mode;
}

export interface PluginThemeProviderProps {
  children: ReactNode;
  /** Emotion cache key. Keep unique per plugin to avoid style collisions with the host. */
  cacheKey?: string;
  /** CSP nonce from the host (PluginApi.cssNonce) so injected <style> tags are allowed. */
  nonce?: string;
}

// Wraps plugin UI in a namespaced Emotion cache + a theme bound to host tokens.
// Deliberately renders no CssBaseline: the host owns document-level globals.
export const PluginThemeProvider = ({
  children,
  cacheKey = 'openeverest-plugin',
  nonce,
}: PluginThemeProviderProps) => {
  const mode = useHostColorMode();
  const cache = useMemo(
    () => createCache({ key: cacheKey, nonce, prepend: true }),
    [cacheKey, nonce]
  );
  // Rebuilt whenever the host colour scheme flips, since every token is read
  // from the computed CSS variables at that moment.
  const theme = useMemo(() => {
    const typography = hostTypography();
    const shape = hostShape();
    return createTheme({
      palette: hostPalette(mode),
      ...(typography ? { typography } : {}),
      ...(shape ? { shape } : {}),
    });
  }, [mode]);

  return (
    <CacheProvider value={cache}>
      <ThemeProvider theme={theme}>{children}</ThemeProvider>
    </CacheProvider>
  );
};
