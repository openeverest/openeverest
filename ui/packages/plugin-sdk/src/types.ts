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
  /** Optional icon URL (absolute or relative path to a PNG/SVG image). */
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
  /**
   * Optional list of database engine types this tab applies to.
   * Values match `spec.engine.type` on the DatabaseCluster CR:
   * `"postgresql"`, `"psmdb"`, `"pxc"`.
   * When omitted (or empty) the tab is shown for all engine types.
   *
   * @example providers: ["postgresql"]        // PostgreSQL only
   * @example providers: ["psmdb", "pxc"]      // MongoDB and MySQL only
   */
  providers?: string[];
}

/** A context-menu action in the databases table row. */
export interface ClusterActionExtension {
  type: "clusterAction";
  /** Action label text shown in the menu. */
  label: string;
  /** The component rendered when the action is triggered. Receives ClusterActionProps. */
  component: ComponentType<ClusterActionProps>;
  /**
   * Optional engine type filter. When set, the action is only shown for clusters
   * whose `spec.engine.type` matches one of the listed values.
   * Omit to show for all engine types.
   */
  providers?: string[];
}

/** A widget card on the cluster overview page. */
export interface ClusterCardExtension {
  type: "clusterCard";
  /** Card title. */
  label: string;
  /** The component rendered inside the card. Receives ClusterCardProps. */
  component: ComponentType<ClusterCardProps>;
  /**
   * Optional engine type filter. When set, the card is only shown for clusters
   * whose `spec.engine.type` matches one of the listed values.
   * Omit to show for all engine types.
   */
  providers?: string[];
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

/**
 * A collapsible section in the create-instance wizard.
 * Lets infrastructure plugins collect configuration during instance creation.
 */
export interface InstanceCreateFormSectionExtension {
  type: "instanceCreateFormSection";
  /** Section heading label. */
  label: string;
  /** The component rendered inside the collapsible section. */
  component: ComponentType<InstanceCreateFormSectionProps>;
  /**
   * Optional engine type filter. When set, the section is only shown for
   * instances whose provider matches one of the listed values.
   * Omit to show for all engine types.
   */
  providers?: string[];
}

/**
 * A collapsible section in the edit-instance / instance detail page.
 * Lets infrastructure plugins expose configuration editing for existing instances.
 */
export interface InstanceEditFormSectionExtension {
  type: "instanceEditFormSection";
  /** Section heading label. */
  label: string;
  /** The component rendered inside the collapsible section. */
  component: ComponentType<InstanceEditFormSectionProps>;
  /**
   * Optional engine type filter. When set, the section is only shown for
   * instances whose provider matches one of the listed values.
   * Omit to show for all engine types.
   */
  providers?: string[];
}

/** Union of all supported extension types. */
export type Extension =
  | RouteExtension
  | SidebarExtension
  | ClusterDetailTabExtension
  | ClusterActionExtension
  | ClusterCardExtension
  | GlobalDashboardWidgetExtension
  | SettingsPanelExtension
  | InstanceCreateFormSectionExtension
  | InstanceEditFormSectionExtension;

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

/** Props injected into an instanceCreateFormSection component. */
export interface InstanceCreateFormSectionProps {
  /** Current form state (read-only snapshot). */
  formValues: Record<string, unknown>;
  /** Callback to update the plugin's config section. */
  onChange: (config: Record<string, unknown>) => void;
  /** Target namespace for the instance. */
  namespace: string;
}

/** Props injected into an instanceEditFormSection component. */
export interface InstanceEditFormSectionProps {
  /** The existing instance resource. */
  instance: unknown;
  /** Current form state (read-only snapshot). */
  formValues: Record<string, unknown>;
  /** Callback to update the plugin's config section. */
  onChange: (config: Record<string, unknown>) => void;
  /** Namespace the instance lives in. */
  namespace: string;
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
