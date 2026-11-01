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

import { useMemo } from 'react';
import { Alert } from '@mui/material';
import { useFormContext, useWatch } from 'react-hook-form';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { useUiGeneratorContext } from 'components/ui-generator/ui-generator-context';
import {
  WidgetRegistry,
  WidgetRenderer,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';
import { AffinityRuleEditor } from './affinity';
import { buildAffinitySections } from './build-scheduling-sections';
import { toAffinityGroups } from './pod-scheduling-policy.utils';
import { SchedulingPolicyTabs } from './scheduling-policy-tabs';
import { SchedulingComponentContext } from './scheduling-component-context';
import { Messages } from './pod-scheduling-policy-section.messages';

const widgetRegistry: WidgetRegistry = {
  [WidgetType.Affinity]: AffinityRuleEditor,
};

// Renders the podSchedulingPolicy marker: one affinity editor per component the
// provider accepts it for (targets resolved by preprocess), one tab each.
export const PodSchedulingPolicySection: WidgetRenderer = ({ item }) => {
  const { providerObject } = useUiGeneratorContext();
  const { control } = useFormContext();
  const targets = item._widgetTargets;
  const sections = useMemo(
    () => buildAffinitySections(targets ?? []),
    [targets]
  );
  const paths = useMemo(
    () => (targets ?? []).map(({ path }) => path),
    [targets]
  );
  const values = useWatch({ control, name: paths });

  if (!targets?.length) {
    return <Alert severity="info">{Messages.notAvailable}</Alert>;
  }

  const tabs = targets.map(({ key }, index) => ({
    key,
    count: toAffinityGroups(values?.[index]).length,
  }));

  return (
    <SchedulingPolicyTabs tabs={tabs}>
      {(activeKey) => (
        <SchedulingComponentContext.Provider value={{ component: activeKey }}>
          <UIGenerator
            sectionKey={activeKey}
            sections={sections}
            providerObject={providerObject}
            widgetRegistry={widgetRegistry}
          />
        </SchedulingComponentContext.Provider>
      )}
    </SchedulingPolicyTabs>
  );
};
