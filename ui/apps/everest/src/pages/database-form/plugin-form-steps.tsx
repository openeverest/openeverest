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

import React, { useMemo } from 'react';
import { useFormContext } from 'react-hook-form';
import type { ComponentType } from 'react';
import type {
  InstanceCreateStepExtension,
  InstanceCreateStepSummaryProps,
  InstanceEditStepExtension,
  InstanceEditStepSummaryProps,
} from '@openeverest/plugin-sdk';
import { usePlugins } from 'contexts/plugins';
import type { StepDefinition } from 'components/ui-generator/form-engine';

interface UsePluginFormStepsArgs {
  /** Whether this is the create wizard (true) or the edit wizard (false). */
  isCreate: boolean;
  /** Target namespace. Plugin steps only render when the plugin is enabled here. */
  namespace: string;
  /** Database engine type (e.g. "psmdb", "pxc", "postgresql"). Drives the provider filter. */
  engineType?: string;
  /** Existing instance (edit mode only). */
  instance?: unknown;
  /**
   * Live ref to per-plugin config blobs, used to seed plugin component state
   * on remount when the user navigates back to the step and to power the
   * right-sidebar summary view.
   */
  initialConfigsRef: React.MutableRefObject<
    Record<string, Record<string, unknown>>
  >;
  /** Called when a plugin step pushes a new config blob. */
  onConfigChange: (pluginName: string, config: Record<string, unknown>) => void;
  /** Called when a plugin step reports its validity. */
  onValidityChange: (pluginName: string, valid: boolean) => void;
}

/**
 * A plugin step summary descriptor rendered in the right-sidebar
 * "Database summary" preview. One entry per plugin that registered an
 * `instanceCreateStep` / `instanceEditStep` extension visible for the
 * current namespace + engine-type.
 */
export interface PluginStepSummary {
  /** Wizard step ID (`plugin:<pluginName>`); used by the preview's Edit link. */
  stepId: string;
  /** Plugin name (matches the Plugin CR name). */
  pluginName: string;
  /** Step label (also used as the summary section title). */
  label: string;
  /**
   * Optional plugin-provided summary component. When omitted the host
   * renders a default "Configured" / "Not configured" placeholder.
   */
  summaryComponent?: ComponentType<
    InstanceCreateStepSummaryProps | InstanceEditStepSummaryProps
  >;
}

export interface PluginFormStepsResult {
  /** Step definitions to append to the wizard. */
  steps: StepDefinition[];
  /** Summary descriptors for the right-sidebar preview, in step order. */
  summaries: PluginStepSummary[];
}

/**
 * Builds wizard `StepDefinition`s for plugins that registered an
 * `instanceCreateStep` (or `instanceEditStep`) extension.
 *
 * Filters by namespace enablement and engine-type, sorts alphabetically by
 * plugin name for deterministic ordering across renders, and wraps each
 * plugin component so it receives the host-managed props.
 */
export const usePluginFormSteps = ({
  isCreate,
  namespace,
  engineType,
  instance,
  initialConfigsRef,
  onConfigChange,
  onValidityChange,
}: UsePluginFormStepsArgs): PluginFormStepsResult => {
  const { plugins } = usePlugins();

  const extensionType = isCreate ? 'instanceCreateStep' : 'instanceEditStep';

  const built = useMemo(() => {
    const matches = plugins.flatMap((p) =>
      p.extensions
        .filter(
          (
            ext
          ): ext is InstanceCreateStepExtension | InstanceEditStepExtension =>
            ext.type === extensionType
        )
        .filter(
          (ext) =>
            !ext.providers?.length ||
            (engineType != null && ext.providers.includes(engineType))
        )
        .map((ext) => ({ pluginName: p.name, ext }))
    );

    // Deterministic ordering: alphabetical by plugin name. Stable across
    // renders so the wizard step indices don't shift unexpectedly.
    matches.sort((a, b) => a.pluginName.localeCompare(b.pluginName));

    return matches.map(
      ({
        pluginName,
        ext,
      }): {
        step: StepDefinition;
        summary: PluginStepSummary;
      } => {
        const PluginComponent = ext.component as unknown as React.ComponentType<
          Record<string, unknown>
        >;

        const StepWrapper: React.FC = () => {
          const { getValues } = useFormContext();

          const handleChange = (config: Record<string, unknown>) => {
            onConfigChange(pluginName, config);
          };
          const handleValidity = (valid: boolean) => {
            onValidityChange(pluginName, valid);
          };

          const props: Record<string, unknown> = {
            formValues: getValues() as Record<string, unknown>,
            initialConfig: initialConfigsRef.current[pluginName],
            onChange: handleChange,
            onValidityChange: handleValidity,
            namespace,
          };
          if (!isCreate) {
            props.instance = instance;
          }

          return React.createElement(PluginComponent, props);
        };
        StepWrapper.displayName = `PluginStep(${pluginName})`;

        const step: StepDefinition = {
          id: `plugin:${pluginName}`,
          kind: 'plugin',
          pluginName,
          optional: ext.optional,
          label: ext.label,
          description: ext.description,
          component: StepWrapper,
          fields: [],
        };

        const summary: PluginStepSummary = {
          stepId: step.id,
          pluginName,
          label: ext.label,
          summaryComponent:
            ext.summaryComponent as PluginStepSummary['summaryComponent'],
        };

        return { step, summary };
      }
    );
  }, [
    plugins,
    extensionType,
    engineType,
    namespace,
    instance,
    isCreate,
    initialConfigsRef,
    onConfigChange,
    onValidityChange,
  ]);

  return useMemo(
    () => ({
      steps: built.map((b) => b.step),
      summaries: built.map((b) => b.summary),
    }),
    [built]
  );
};
