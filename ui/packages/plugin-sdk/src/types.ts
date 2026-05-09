import type { ComponentType } from "react";

// ---------------------------------------------------------------------------
// Extension types — what a plugin can contribute to the host
// ---------------------------------------------------------------------------

/** A page route rendered inside the host shell at /plugins/<pluginName>/* */
export interface RouteExtension {
  type: "route";
  /** Human-readable label (used in breadcrumbs, page title, etc.) */
  label: string;
  /** The React component rendered for this route. Receives PluginRouteProps. */
  component: ComponentType<PluginRouteProps>;
}

/** A sidebar navigation item added to the host's drawer. */
export interface SidebarExtension {
  type: "sidebarItem";
  /** Text shown in the sidebar */
  label: string;
  /** Optional MUI icon name (future: allow custom icon components) */
  icon?: string;
}

/** An extra tab on the database instance detail page. */
export interface ClusterDetailTabExtension {
  type: "clusterDetailTab";
  /** Tab label text. */
  label: string;
  /** URL-safe slug used as the tab route segment (e.g. "query" → /databases/:ns/:name/query). */
  path: string;
  /** The component rendered when the tab is active. Receives ClusterDetailTabProps. */
  component: ComponentType<ClusterDetailTabProps>;
}

/** A context-menu action in the databases table row. */
export interface ClusterActionExtension {
  type: "clusterAction";
  /** Action label text shown in the menu. */
  label: string;
  /** The component rendered when the action is triggered. Receives ClusterActionProps. */
  component: ComponentType<ClusterActionProps>;
}

/** A widget card on the cluster overview page. */
export interface ClusterCardExtension {
  type: "clusterCard";
  /** Card title. */
  label: string;
  /** The component rendered inside the card. Receives ClusterCardProps. */
  component: ComponentType<ClusterCardProps>;
}

/** A card on the home / dashboard page. */
export interface GlobalDashboardWidgetExtension {
  type: "globalDashboardWidget";
  /** Widget card title. */
  label: string;
  /** The component rendered inside the widget. Receives GlobalDashboardWidgetProps. */
  component: ComponentType<GlobalDashboardWidgetProps>;
}

/** An extra tab inside the Settings page. */
export interface SettingsPanelExtension {
  type: "settingsPanel";
  /** Tab label text. */
  label: string;
  /** URL-safe slug used as the settings tab route segment. */
  path: string;
  /** The component rendered when the tab is active. Receives SettingsPanelProps. */
  component: ComponentType<SettingsPanelProps>;
}

/** Union of all supported extension types. */
export type Extension =
  | RouteExtension
  | SidebarExtension
  | ClusterDetailTabExtension
  | ClusterActionExtension
  | ClusterCardExtension
  | GlobalDashboardWidgetExtension
  | SettingsPanelExtension;

// ---------------------------------------------------------------------------
// Props passed to plugin-provided components
// ---------------------------------------------------------------------------

/** Props injected by the host into a plugin's route component. */
export interface PluginRouteProps {
  /** The registered name of the plugin (matches the Plugin CR name). */
  pluginName: string;
  /** The wildcard sub-path after /plugins/<pluginName>/. */
  subPath?: string;
}

/** Props injected into a clusterDetailTab component. */
export interface ClusterDetailTabProps {
  /** The Instance resource. */
  cluster: unknown;
  /** Kubernetes namespace the instance lives in. */
  namespace: string;
  /** Instance name. */
  instanceName: string;
}

/** Props injected into a clusterAction component. */
export interface ClusterActionProps {
  /** The Instance resource. */
  cluster: unknown;
  /** Kubernetes namespace. */
  namespace: string;
  /** Callback to close the action modal/popover. */
  onClose: () => void;
}

/** Props injected into a clusterCard component. */
export interface ClusterCardProps {
  /** The Instance resource. */
  cluster: unknown;
  /** Kubernetes namespace. */
  namespace: string;
}

/** Props injected into a globalDashboardWidget component. */
export interface GlobalDashboardWidgetProps {
  /** Namespaces the current user has access to. */
  namespaces: string[];
}

/** Props injected into a settingsPanel component. */
export interface SettingsPanelProps {
  /** Currently logged-in user identifier. */
  currentUser: string;
}

// ---------------------------------------------------------------------------
// Plugin API — the object passed to register()
// ---------------------------------------------------------------------------

/** The API object provided to a plugin's register() function by the host. */
export interface PluginApi {
  /**
   * The host's React instance. Plugins MUST use this instead of importing
   * their own React to avoid duplicate-React issues with hooks.
   */
  React: typeof import("react");

  /** Register a UI extension with the host. */
  registerExtension(extension: Extension): void;

  /**
   * Make an authenticated API call through the host's proxy.
   * Equivalent to fetch(`/v1/plugins/${pluginName}${path}`, init).
   * The host automatically attaches the auth token.
   */
  fetch(path: string, init?: RequestInit): Promise<Response>;
}

// ---------------------------------------------------------------------------
// Plugin entry point contract
// ---------------------------------------------------------------------------

/**
 * The function signature that a plugin module must export.
 * The host calls this after dynamically importing the plugin's bundle.
 *
 * @example
 * ```ts
 * import type { PluginRegisterFn, PluginApi } from '@openeverest/plugin-sdk';
 *
 * const register: PluginRegisterFn = (api) => {
 *   const React = api.React;
 *   const MyPage = () => React.createElement('h1', null, 'Hello!');
 *
 *   api.registerExtension({ type: 'sidebarItem', label: 'My Plugin' });
 *   api.registerExtension({ type: 'route', label: 'My Plugin', component: MyPage });
 * };
 *
 * export default register;
 * ```
 */
export type PluginRegisterFn = (api: PluginApi) => void;
