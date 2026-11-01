# @openeverest/plugin-theme

Makes an OpenEverest plugin's own MUI look like the host: palette, typography,
corner radius and light/dark mode, switching live.

The host shares only React with plugins. Each plugin bundles its own MUI, so a
host MUI upgrade never breaks an already-built plugin. This package bridges the
two: it reads the host's `--everest-*` design tokens (CSS variables) and builds
a MUI theme from them with the plugin's MUI.

Requires an OpenEverest host that publishes the tokens (v2.0.0-dev.4 or later).
Tokens the host doesn't publish fall back to MUI defaults.

## Install

```sh
npm install @openeverest/plugin-theme @mui/material @emotion/react @emotion/styled @emotion/cache
npm install -D @openeverest/plugin-sdk react react-dom @types/react
```

| Peer | Range |
|---|---|
| `@mui/material` | `^5.15.0 \|\| ^6.0.0 \|\| ^7.0.0 \|\| ^9.0.0` |
| `@emotion/react`, `@emotion/cache` | `^11.11.0` |
| `react`, `react-dom` | `^18.0.0` |

Bundle MUI, Emotion and this package into the plugin. Keep `react`,
`react-dom` and `react/jsx-runtime` external: the host import map provides
them.

## Usage

Wrap every component you register with the host:

```tsx
import type { ReactNode } from 'react';
import { Button } from '@mui/material';
import { PluginThemeProvider } from '@openeverest/plugin-theme';
import type { PluginApi, PluginRegisterFn } from '@openeverest/plugin-sdk';

let pluginApi: PluginApi;

const PluginRoot = ({ children }: { children: ReactNode }) => (
  <PluginThemeProvider cacheKey="sql-explorer" nonce={pluginApi.cssNonce}>
    {children}
  </PluginThemeProvider>
);

const Page = () => (
  <PluginRoot>
    <Button variant="contained">Run query</Button>
  </PluginRoot>
);

const register: PluginRegisterFn = (api) => {
  pluginApi = api;
  api.registerExtension({ type: 'route', label: 'SQL Explorer', component: Page });
};

export default register;
```

### `PluginThemeProvider`

| Prop | Required | Value |
|---|---|---|
| `cacheKey` | Yes | Emotion cache key, also the class-name prefix of the plugin's styles. Use the plugin name; lowercase letters and `-` only. Must be unique across plugins and not `percona-css` (the host's). |
| `nonce` | Under the host CSP | `api.cssNonce`. Without it the browser blocks the plugin's styles. |
| `children` | Yes | The plugin UI. |

It renders no `CssBaseline`: document-level styles belong to the host. Don't
wrap plugin UI in your own `ThemeProvider`, and don't enable MUI
`cssVariables` (it writes `--mui-*` variables to `:root`).

### `useHostColorMode(): 'light' | 'dark'`

The host's current mode; re-renders on change. For code outside the MUI theme,
such as chart libraries or canvas drawing.

## Host tokens

The host publishes CSS custom properties on `:root`:

- `--everest-color-<name>` and `--everest-color-<name>-<shade>`: palette, text,
  background and divider colours;
- `--everest-font-<variant>-<property>`: typography per MUI variant;
- `--everest-radius`: corner radius, unitless pixels.

The mode is `data-everest-color-scheme="light" | "dark"` on `<html>`. Code that
doesn't use MUI can read these directly, e.g.
`color: var(--everest-color-text-secondary)`.

The full list is in
[`everest-tokens.ts`](https://github.com/openeverest/openeverest/blob/main/ui/packages/design/src/theme-context-provider/everest-tokens.ts).
Variables are only ever added; renaming or removing one is a breaking host
change.

## Not shared yet

The host theme's component overrides (e.g. the pill-shaped `Button`), custom
typography variants, shadows, the grey scale and action states.

See the [plugin architecture spec](https://github.com/openeverest/specs/blob/main/specs/003-generic-plugins.md)
(§8.1, §9.3–9.5) for bundle requirements and compatibility rules.
