# @openeverest/plugin-sdk

TypeScript types and the `register` contract for OpenEverest plugin frontends.
The package has no runtime code; install it as a dev dependency.

```sh
npm install -D @openeverest/plugin-sdk
```

## Entry point

A plugin frontend is a single ES module whose default export is a
`PluginRegisterFn`. The host imports it and calls it with a `PluginApi`:

```tsx
import type { PluginRegisterFn } from '@openeverest/plugin-sdk';
import { QueryTab } from './query-tab';

const register: PluginRegisterFn = (api) => {
  api.registerExtension({
    type: 'clusterDetailTab',
    label: 'Query',
    path: 'query',
    providers: ['provider-percona-postgresql'],
    component: QueryTab,
  });
};

export default register;
```

## `PluginApi`

| Member | Description |
|---|---|
| `registerExtension(extension)` | Adds a UI contribution: `route`, `sidebarItem`, `clusterDetailTab`, `clusterAction`, `clusterCard`, `globalDashboardWidget`, `settingsPanel`, `instanceCreateFormSection`, `instanceEditFormSection`. |
| `fetch(path, init?)` | Authenticated call to the plugin's own backend through the host proxy. |
| `basePath` | The proxy path the plugin is served under, for asset URLs. |
| `cssNonce` | CSP nonce for `<style>` tags; pass it to `PluginThemeProvider` from `@openeverest/plugin-theme`. |
| `hostVersion` | Host version, `"dev"` when unknown. |
| `uiContractVersion` | The shared React major, e.g. `"18"`. |
| `React` | The host React, for plugins built without a bundler. |

`cssNonce`, `hostVersion` and `uiContractVersion` are new in 0.4.0 and require
an OpenEverest host v2.0.0-dev.4 or later.

## Building a plugin

Keep `react`, `react-dom` and `react/jsx-runtime` external: the host import map
resolves them to its own React. Bundle everything else, including your UI
library. For a native look with MUI, use
[`@openeverest/plugin-theme`](https://www.npmjs.com/package/@openeverest/plugin-theme).

Bundle requirements, the host CSP and compatibility ranges are described in the
[plugin architecture spec](https://github.com/openeverest/specs/blob/main/specs/003-generic-plugins.md)
(§8.1, §9).
