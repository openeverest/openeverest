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
  type TypographyVariantsOptions,
} from '@mui/material';

// One variant's style options (fontFamily/fontSize/textTransform/...), derived
// from MUI's own type so we never re-declare or cast it.
type VariantStyle = NonNullable<TypographyVariantsOptions['button']>;

// The host publishes its active color scheme here (see @percona/design
// ThemeContextProvider). Plugins can't read the host React context because
// they bundle their own MUI, so they observe this attribute instead.
const HOST_COLOR_SCHEME_ATTR = 'data-everest-color-scheme';

// Reads a host-owned `--everest-*` CSS variable's *computed* value. We resolve
// to concrete values rather than passing `var(...)` into the theme, because
// MUI's color functions (alpha/darken/lighten) can't parse `var()` strings
// (error #9). The `--everest-*` names are the host's stable design contract, so
// a host MUI upgrade that renames MUI's internal `--mui-*` vars can't break us.
function readVar(name: string): string {
  if (typeof document === 'undefined') {
    return '';
  }
  return getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim();
}

function colorFromHost(name: string): SimplePaletteColorOptions | undefined {
  const main = readVar(`--everest-color-${name}-main`);
  if (!main) {
    return undefined;
  }
  const color: SimplePaletteColorOptions = { main };
  const dark = readVar(`--everest-color-${name}-dark`);
  const light = readVar(`--everest-color-${name}-light`);
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

  const textPrimary = readVar('--everest-color-text-primary');
  if (textPrimary) {
    const secondaryText = readVar('--everest-color-text-secondary');
    const disabledText = readVar('--everest-color-text-disabled');
    palette.text = {
      primary: textPrimary,
      ...(secondaryText && { secondary: secondaryText }),
      ...(disabledText && { disabled: disabledText }),
    };
  }

  const bgDefault = readVar('--everest-color-background-default');
  if (bgDefault) {
    const paper = readVar('--everest-color-background-paper');
    palette.background = {
      default: bgDefault,
      ...(paper && { paper }),
    };
  }

  const divider = readVar('--everest-color-divider');
  if (divider) {
    palette.divider = divider;
  }

  return palette;
}

// The typography variants the host publishes as `--everest-font-*` tokens.
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

// CSS `text-transform` keywords the host may publish. Narrowing against this set
// lets us assign to the typed `textTransform` field without an `as` cast.
const TEXT_TRANSFORMS = [
  'none',
  'capitalize',
  'uppercase',
  'lowercase',
] as const;

function isTextTransform(
  value: string
): value is (typeof TEXT_TRANSFORMS)[number] {
  return (TEXT_TRANSFORMS as readonly string[]).includes(value);
}

// Reads one variant's per-property `--everest-font-<variant>-*` tokens. The host
// publishes each property separately, so there is nothing to parse and both
// textTransform and letterSpacing cross the bridge (a `font` shorthand can't
// carry them). Missing tokens are simply omitted.
function fontFromHost(variant: string): VariantStyle | undefined {
  const style: VariantStyle = {};

  const family = readVar(`--everest-font-${variant}-family`);
  if (family) {
    style.fontFamily = family;
  }
  const size = readVar(`--everest-font-${variant}-size`);
  if (size) {
    style.fontSize = size;
  }
  const weight = readVar(`--everest-font-${variant}-weight`);
  if (weight) {
    const numeric = Number(weight);
    style.fontWeight = Number.isNaN(numeric) ? weight : numeric;
  }
  const lineHeight = readVar(`--everest-font-${variant}-line-height`);
  if (lineHeight) {
    style.lineHeight = lineHeight;
  }
  const letterSpacing = readVar(`--everest-font-${variant}-letter-spacing`);
  if (letterSpacing) {
    style.letterSpacing = letterSpacing;
  }
  const transform = readVar(`--everest-font-${variant}-transform`);
  if (isTextTransform(transform)) {
    style.textTransform = transform;
  }

  return Object.keys(style).length ? style : undefined;
}

// Builds the typography scale from host tokens. Variants the host doesn't
// publish simply fall back to the plugin's own MUI defaults.
function hostTypography(): TypographyVariantsOptions | undefined {
  const typography: TypographyVariantsOptions = {};
  for (const variant of TYPOGRAPHY_VARIANTS) {
    const style = fontFromHost(variant);
    if (style) {
      typography[variant] = style;
    }
  }
  // Give unlisted variants the host's body face rather than MUI's default.
  const bodyFamily = typography.body1?.fontFamily;
  if (bodyFamily) {
    typography.fontFamily = bodyFamily;
  }
  return Object.keys(typography).length ? typography : undefined;
}

function hostShape(): ThemeOptions['shape'] {
  const radius = parseFloat(readVar('--everest-radius'));
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
  /**
   * Emotion cache key. Required and must be unique per plugin (e.g. the plugin
   * name) so two plugins never share a cache and clobber each other's styles.
   */
  cacheKey: string;
  /** CSP nonce from the host (PluginApi.cssNonce) so injected <style> tags are allowed. */
  nonce?: string;
}

// Wraps plugin UI in a namespaced Emotion cache + a theme bound to host tokens.
// Deliberately renders no CssBaseline: the host owns document-level globals.
export const PluginThemeProvider = ({
  children,
  cacheKey,
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
