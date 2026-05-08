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

/** Union of all supported extension types. */
export type Extension = RouteExtension | SidebarExtension;

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
