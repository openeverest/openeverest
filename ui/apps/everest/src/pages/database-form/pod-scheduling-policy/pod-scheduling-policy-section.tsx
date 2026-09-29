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

import { useMemo, useState } from 'react';
import {
  Alert,
  Badge,
  Box,
  Switch,
  Tab,
  Tabs,
  Typography,
} from '@mui/material';
import { useFormContext, useWatch } from 'react-hook-form';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import RoundedBox from 'components/rounded-box';
import {
  WidgetRegistry,
  WidgetRenderer,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';
import { DbWizardFormFields } from 'consts';
import { deriveSchedulingSupport } from 'utils/pod-scheduling-policy';
import { AffinityRuleEditor } from './affinity';
import {
  buildAffinitySections,
  schedulingPolicyPath,
} from './build-scheduling-sections';
import { Messages } from './pod-scheduling-policy-section.messages';
import { useDatabaseFormContext } from '../database-form-context';

const widgetRegistry: WidgetRegistry = {
  [WidgetType.Affinity]: AffinityRuleEditor,
};

// Domain orchestrator: renders the scheduling marker. Derives which components
// accept affinity for the selected topology and renders the engine per
// component. Layout (single vs tabbed) lives here so the engine stays
// domain-free. The marker carries no field, so name/item are unused.
export const PodSchedulingPolicySection: WidgetRenderer = () => {
  const { providerObject } = useDatabaseFormContext();
  const { control } = useFormContext();
  const topology = useWatch({ control, name: DbWizardFormFields.topology });

  const sections = useMemo(
    () =>
      buildAffinitySections(deriveSchedulingSupport(providerObject, topology)),
    [providerObject, topology]
  );
  const components = useMemo(() => Object.keys(sections), [sections]);

  const affinityPaths = useMemo(
    () =>
      components.map((component) =>
        schedulingPolicyPath(component, 'affinity')
      ),
    [components]
  );
  const affinityValues = useWatch({ control, name: affinityPaths });

  const [selected, setSelected] = useState<string>();
  const activeComponent =
    selected && components.includes(selected) ? selected : components[0];

  // MOCK (temporary, hardcoded): v1-style bordered card with a cosmetic
  // expand/collapse toggle that carries no form value.
  const [expanded, setExpanded] = useState(true);
  const cardTitle = (
    <Box
      sx={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'flex-start',
      }}
    >
      <Box>
        <Typography variant="sectionHeading">{Messages.title}</Typography>
        <Typography variant="body2" color="text.secondary">
          {Messages.description}
        </Typography>
      </Box>
      <Switch
        checked={expanded}
        onChange={(_, value) => setExpanded(value)}
        data-testid="pod-scheduling-toggle"
      />
    </Box>
  );

  // A0: the provider placed the marker but no component accepts affinity for
  // this topology (an explicit supportedFields exclusion). Nothing to edit.
  if (components.length === 0) {
    return (
      <RoundedBox title={cardTitle}>
        {expanded && (
          <Alert severity="info" sx={{ mt: 2 }}>
            {Messages.notAvailable(topology)}
          </Alert>
        )}
      </RoundedBox>
    );
  }

  const generator = (
    <UIGenerator
      sectionKey={activeComponent}
      sections={sections}
      providerObject={providerObject}
      widgetRegistry={widgetRegistry}
    />
  );

  const componentHasRules = (index: number): boolean => {
    const value = affinityValues?.[index];
    return (
      !!value && typeof value === 'object' && Object.keys(value).length > 0
    );
  };

  return (
    <RoundedBox title={cardTitle}>
      {expanded && (
        <Box sx={{ mt: 2 }}>
          <Tabs
            value={activeComponent}
            onChange={(_, value) => setSelected(value)}
            variant="scrollable"
            sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}
          >
            {components.map((component, index) => (
              <Tab
                key={component}
                value={component}
                label={
                  <Badge
                    color="primary"
                    variant="dot"
                    invisible={!componentHasRules(index)}
                    sx={{ '& .MuiBadge-badge': { right: -8, top: 6 } }}
                  >
                    {component}
                  </Badge>
                }
                data-testid={`scheduling-component-tab-${component}`}
              />
            ))}
          </Tabs>
          {generator}
        </Box>
      )}
    </RoundedBox>
  );
};
