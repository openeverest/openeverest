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

import type { Theme } from '@mui/material';

// The `--everest-*` custom properties published on :root are the *public design
// contract* plugins build against (they bundle their own MUI and read these
// instead of MUI's internal `--mui-*` names, which a host MUI upgrade could
// rename). Keep the names below stable; changing one is a breaking change for
// every plugin. The reader lives in `@openeverest/ui-lib` (plugin-ui-lib).

const PALETTE_COLORS = [
  'primary',
  'secondary',
  'error',
  'warning',
  'info',
  'success',
] as const;

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

// Publishes the host design tokens as `--everest-*` CSS variables on :root so
// plugins inherit the palette, typography, shape and dark mode through the CSS
// cascade without reading the host React context or MUI internals.
export function writeEverestTokens(theme: Theme): void {
  if (typeof document === 'undefined') {
    return;
  }
  const root = document.documentElement;
  const set = (name: string, value: string | number | undefined): void => {
    if (value === undefined || value === null || value === '') {
      return;
    }
    root.style.setProperty(name, String(value));
  };

  for (const name of PALETTE_COLORS) {
    const color = theme.palette[name];
    if (!color) {
      continue;
    }
    set(`--everest-color-${name}-main`, color.main);
    set(`--everest-color-${name}-dark`, color.dark);
    set(`--everest-color-${name}-light`, color.light);
  }

  set('--everest-color-text-primary', theme.palette.text?.primary);
  set('--everest-color-text-secondary', theme.palette.text?.secondary);
  set('--everest-color-text-disabled', theme.palette.text?.disabled);
  set('--everest-color-background-default', theme.palette.background?.default);
  set('--everest-color-background-paper', theme.palette.background?.paper);
  set('--everest-color-divider', theme.palette.divider);
  set('--everest-radius', theme.shape?.borderRadius);

  for (const variant of TYPOGRAPHY_VARIANTS) {
    const style = theme.typography[variant];
    if (!style) {
      continue;
    }
    // Each property is published separately so plugins never parse a `font`
    // shorthand, and textTransform/letterSpacing (which the shorthand can't
    // carry) cross the boundary too.
    set(`--everest-font-${variant}-family`, style.fontFamily);
    set(`--everest-font-${variant}-size`, style.fontSize);
    set(`--everest-font-${variant}-weight`, style.fontWeight);
    set(`--everest-font-${variant}-line-height`, style.lineHeight);
    set(`--everest-font-${variant}-letter-spacing`, style.letterSpacing);
    set(`--everest-font-${variant}-transform`, style.textTransform);
  }
}
