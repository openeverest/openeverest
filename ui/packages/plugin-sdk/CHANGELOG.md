# Changelog

## 0.4.0

Requires an OpenEverest host v2.0.0-dev.4 or later for the new `PluginApi`
fields. On an older host they are `undefined`, and without `cssNonce` the
host's CSP blocks the plugin's styles. Declare
`compatibleHostVersions: ">=2.0.0-dev.4 <3.0.0"` on the `Plugin` CR so an older
host rejects the plugin instead.

- Added `cssNonce`, `hostVersion` and `uiContractVersion` to `PluginApi`.
- The peer dependency is now `@types/react` (optional) instead of `react`: the
  package is types only and has no runtime.
- Clarified `PluginApi.React`: bundled plugins import `react`, which the host
  import map resolves to the host React; `api.React` is for plugins built
  without a bundler.
- Added a README.

Upgrading from 0.3.x needs no code changes.
