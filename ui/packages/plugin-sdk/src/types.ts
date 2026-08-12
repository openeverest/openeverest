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
   * Optional list of Provider names this tab applies to.
   * Values match `spec.providerRef.name` on the Instance CR, e.g.
   * `"provider-percona-postgresql"`, `"percona-server-mongodb"`,
   * `"percona-xtradb-cluster"`. Provider names are not prefix-consistent
   * across providers; check the target provider's own definition.
   * When omitted (or empty) the tab is shown for all providers.
   *
   * @example providers: ["provider-percona-postgresql"]  // PostgreSQL only
   * @example providers: ["percona-server-mongodb", "percona-xtradb-cluster"]  // MongoDB and MySQL only
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
   * Optional Provider filter. When set, the action is only shown for clusters
   * whose `spec.providerRef.name` matches one of the listed values.
   * Omit to show for all providers.
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
   * Optional Provider filter. When set, the card is only shown for clusters
   * whose `spec.providerRef.name` matches one of the listed values.
   * Omit to show for all providers.
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
   * Optional Provider filter. When set, the section is only shown for
   * instances whose `spec.providerRef.name` matches one of the listed values.
   * Omit to show for all providers.
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
   * Optional Provider filter. When set, the section is only shown for
   * instances whose `spec.providerRef.name` matches one of the listed values.
   * Omit to show for all providers.
   */
  providers?: string[];
}

/**
 * A full wizard step appended after the provider-defined steps in the
 * create-instance flow. Use this when a plugin needs its own dedicated page
 * (its own context, multi-field layout, or Next-button gating). For small,
 * unobtrusive add-ons prefer `instanceCreateFormSection`.
 *
 * Plugin steps are always appended at the end of the wizard, after the
 * provider-encoded steps. The host orders them deterministically by
 * `(installedAt, pluginName)`.
 */
export interface InstanceCreateStepExtension {
  type: "instanceCreateStep";
  /** Step label shown in the stepper. */
  label: string;
  /** Optional longer description shown as the step header subtitle. */
  description?: string;
  /** Optional MUI icon component shown next to the label. */
  icon?: ComponentType;
  /**
   * Optional engine-type filter. When set, the step is only shown for
   * instances whose provider matches one of the listed values.
   * Omit to show for all engine types.
   */
  providers?: string[];
  /**
   * If true, the wizard's Next/Submit button is enabled even when the plugin
   * never calls `onValidityChange`. Set to true for steps the user may skip.
   */
  optional?: boolean;
  /** The component rendered when this step is active. */
  component: ComponentType<InstanceCreateStepProps>;
  /**
   * Optional read-only summary rendered in the wizard's right-side
   * "Database summary" preview drawer. Receives the same `formValues` and
   * the most recent `config` blob the plugin pushed via `onChange`.
   * When omitted the host shows a default "Configured" / "Not configured"
   * placeholder so the step is still listed in the summary.
   */
  summaryComponent?: ComponentType<InstanceCreateStepSummaryProps>;
}

/**
 * A full wizard step appended after the provider-defined steps in the
 * edit-instance flow. See `InstanceCreateStepExtension` for the full contract.
 */
export interface InstanceEditStepExtension {
  type: "instanceEditStep";
  label: string;
  description?: string;
  icon?: ComponentType;
  providers?: string[];
  optional?: boolean;
  component: ComponentType<InstanceEditStepProps>;
  summaryComponent?: ComponentType<InstanceEditStepSummaryProps>;
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
  | InstanceEditFormSectionExtension
  | InstanceCreateStepExtension
  | InstanceEditStepExtension;

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

/**
 * Props injected into an `instanceCreateStep` component.
 *
 * Note on remount semantics: when the user navigates back to this step,
 * the component is remounted. To preserve user input across navigation,
 * seed local state from `initialConfig` on mount — it carries the last
 * value the plugin pushed to `onChange`.
 */
export interface InstanceCreateStepProps {
  /** Current form state across all preceding steps (read-only snapshot). */
  formValues: Record<string, unknown>;
  /**
   * The last config blob the plugin pushed via `onChange`, or `undefined`
   * if the plugin has not configured anything yet. Use to seed state on
   * mount so input is not lost when the user navigates back to this step.
   */
  initialConfig?: Record<string, unknown>;
  /** Callback to update the plugin's config blob. */
  onChange: (config: Record<string, unknown>) => void;
  /**
   * Callback to gate the wizard's Next/Submit button while the user is on
   * this step. Pass `true` when the step's input is valid, `false` to block.
   * A step that never calls this is treated as valid (or controlled by
   * `optional: true` on the extension registration).
   */
  onValidityChange: (valid: boolean) => void;
  /** Target namespace for the instance. */
  namespace: string;
}

/** Props injected into an `instanceEditStep` component. */
export interface InstanceEditStepProps {
  /** The existing instance resource. */
  instance: unknown;
  /** Current form state across all preceding steps (read-only snapshot). */
  formValues: Record<string, unknown>;
  /**
   * The last config blob the plugin pushed via `onChange`, or `undefined`
   * if the plugin has not configured anything yet during this edit session.
   */
  initialConfig?: Record<string, unknown>;
  /** Callback to update the plugin's config blob. */
  onChange: (config: Record<string, unknown>) => void;
  /** Callback to gate the wizard's Next/Submit button. */
  onValidityChange: (valid: boolean) => void;
  /** Namespace the instance lives in. */
  namespace: string;
}

/**
 * Props injected into an `instanceCreateStep` summary component. Rendered
 * in the wizard's right-side "Database summary" drawer as a read-only
 * preview of the plugin's contribution.
 */
export interface InstanceCreateStepSummaryProps {
  /** Current form state across all preceding steps (read-only snapshot). */
  formValues: Record<string, unknown>;
  /**
   * The most recent config blob the plugin pushed via `onChange`. Undefined
   * when the user has not visited the plugin step yet.
   */
  config?: Record<string, unknown>;
  /** Target namespace for the instance. */
  namespace: string;
}

/** Props injected into an `instanceEditStep` summary component. */
export interface InstanceEditStepSummaryProps {
  /** The existing instance resource. */
  instance: unknown;
  /** Current form state across all preceding steps (read-only snapshot). */
  formValues: Record<string, unknown>;
  /** The most recent config blob the plugin pushed via `onChange`. */
  config?: Record<string, unknown>;
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
