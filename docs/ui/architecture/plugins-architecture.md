# UI Plugins Architecture

This document explains how plugins integrate with the UI,
which extension points are supported, and where integration boundaries exist.

## Plugin Lifecycle

```mermaid
flowchart TB
  A[PluginProvider start] --> B[GET /v1/plugins]
  B --> C[Plugin descriptors]
  C --> D[Load plugin bundle]
  D --> E[Register plugin API]
  E --> F[Register extension]
  F --> G[Filter by CRD extension points]
  G --> H[Store registrations in PluginContext]
  H --> I[Render through hosts]

  I --> H1[PluginHost\nroute]
  I --> H2[PluginTabHost\nclusterDetailTab]
  I --> H3[PluginSettingsHost\nsettingsPanel]
```

## Extension Points Map (High Level)

```mermaid
flowchart LR
  subgraph EXT[Extension points]
    E1[route]
    E2[sidebarItem]
    E3[clusterDetailTab]
    E4[settingsPanel]
    E5[clusterAction]
    E6[clusterCard]
    E7[globalDashboardWidget]
    E8[instanceCreateFormSection\ninstanceEditFormSection]
  end

  E1 --> R1[plugins/:pluginName/*]
  E2 --> R2[Drawer]
  E3 --> R3[DB details tabs]
  E4 --> R4[Settings tabs]
  E8 --> R5[Database form integration]
```

## Integration Boundaries

1. Plugins are loaded and registered via PluginProvider.
2. Plugin rendering always goes through host components.
3. Plugin component errors are isolated by PluginErrorBoundary.
4. Plugin extensions are limited by extension points declared in CRD.

## Key Files

1. [../../../ui/apps/everest/src/contexts/plugins/plugins.context.tsx](../../../ui/apps/everest/src/contexts/plugins/plugins.context.tsx)
2. [../../../ui/apps/everest/src/components/plugin-host/PluginHost.tsx](../../../ui/apps/everest/src/components/plugin-host/PluginHost.tsx)
3. [../../../ui/apps/everest/src/components/plugin-host/PluginTabHost.tsx](../../../ui/apps/everest/src/components/plugin-host/PluginTabHost.tsx)
4. [../../../ui/apps/everest/src/components/plugin-host/PluginSettingsHost.tsx](../../../ui/apps/everest/src/components/plugin-host/PluginSettingsHost.tsx)
5. [../../../ui/packages/plugin-sdk](../../../ui/packages/plugin-sdk)

## Related Documents

1. [../../process/generic-plugins-design.md](../../process/generic-plugins-design.md)
2. [./everest-app-overview.md](./everest-app-overview.md)

## Metadata

- Owner: UI
- Last updated: 2026-06-11
