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

import React, { useMemo, useCallback } from 'react';
import {
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { usePlugins } from 'contexts/plugins';
import { usePluginsForNamespace } from 'hooks/api/plugins/usePluginsForNamespace';
import type {
  InstanceCreateFormSectionExtension,
  InstanceEditFormSectionExtension,
} from '@openeverest/plugin-sdk';

interface PluginFormSectionsProps {
  /** Current form values snapshot. */
  formValues: Record<string, unknown>;
  /** Target namespace. */
  namespace: string;
  /** Database engine type (provider), e.g. "psmdb", "pxc", "postgresql". */
  engineType?: string;
  /** Existing instance (only for edit mode). */
  instance?: unknown;
  /** Whether this is create mode (true) or edit mode (false). */
  isCreate: boolean;
  /** Callback to collect plugin configs. Keyed by plugin name. */
  onPluginConfigChange: (
    pluginName: string,
    config: Record<string, unknown>
  ) => void;
}

/**
 * Renders collapsible accordion sections for each plugin that registered
 * an `instanceCreateFormSection` or `instanceEditFormSection` extension.
 * Only shows plugins that are installed in the target namespace and match
 * the engine type filter.
 */
export const PluginFormSections: React.FC<PluginFormSectionsProps> = ({
  formValues,
  namespace,
  engineType,
  instance,
  isCreate,
  onPluginConfigChange,
}) => {
  const { plugins } = usePlugins();
  const { data: pluginsEnabledInNamespace } = usePluginsForNamespace(namespace);

  // Set of plugins installed in this namespace (or null = no filter).
  const enabledInNs = pluginsEnabledInNamespace?.length
    ? new Set(pluginsEnabledInNamespace.map((p) => p.name))
    : null;

  const extensionType = isCreate
    ? 'instanceCreateFormSection'
    : 'instanceEditFormSection';

  const sections = useMemo(() => {
    return plugins.flatMap((p) =>
      p.extensions
        .filter(
          (
            ext
          ): ext is
            | InstanceCreateFormSectionExtension
            | InstanceEditFormSectionExtension => ext.type === extensionType
        )
        .filter(
          (ext) =>
            !ext.providers?.length ||
            (engineType != null && ext.providers.includes(engineType))
        )
        .filter(() => enabledInNs === null || enabledInNs.has(p.name))
        .map((ext) => ({ pluginName: p.name, ...ext }))
    );
  }, [plugins, extensionType, engineType, enabledInNs]);

  if (sections.length === 0) return null;

  return (
    <>
      {sections.map((section) => (
        <PluginFormSection
          key={`${section.pluginName}-${section.label}`}
          pluginName={section.pluginName}
          section={section}
          formValues={formValues}
          namespace={namespace}
          instance={instance}
          isCreate={isCreate}
          onConfigChange={onPluginConfigChange}
        />
      ))}
    </>
  );
};

interface PluginFormSectionItemProps {
  pluginName: string;
  section: (
    | InstanceCreateFormSectionExtension
    | InstanceEditFormSectionExtension
  ) & {
    pluginName: string;
  };
  formValues: Record<string, unknown>;
  namespace: string;
  instance?: unknown;
  isCreate: boolean;
  onConfigChange: (pluginName: string, config: Record<string, unknown>) => void;
}

const PluginFormSection: React.FC<PluginFormSectionItemProps> = ({
  pluginName,
  section,
  formValues,
  namespace,
  instance,
  isCreate,
  onConfigChange,
}) => {
  const handleChange = useCallback(
    (config: Record<string, unknown>) => {
      onConfigChange(pluginName, config);
    },
    [pluginName, onConfigChange]
  );

  const props = isCreate
    ? { formValues, onChange: handleChange, namespace }
    : { instance, formValues, onChange: handleChange, namespace };

  return (
    <Accordion defaultExpanded={false} sx={{ mt: 2 }}>
      <AccordionSummary expandIcon={<ExpandMoreIcon />}>
        <Typography variant="subtitle1">{section.label}</Typography>
      </AccordionSummary>
      <AccordionDetails>
        {React.createElement(
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          section.component as React.ComponentType<any>,
          props
        )}
      </AccordionDetails>
    </Accordion>
  );
};
