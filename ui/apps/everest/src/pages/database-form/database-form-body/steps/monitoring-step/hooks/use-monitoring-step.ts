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

import { useCallback, useMemo } from 'react';
import { useMonitoringClassesForProvider } from 'hooks/api/monitoring-classes';
import { MonitoringClass } from 'shared-types/monitoring-destinations.types';
import {
  buildMonitoringParametersSchema,
  writeMonitoringEntries,
} from '../monitoring-step.utils';
import { useMonitoringInstanceForms } from './use-monitoring-instance-forms';

const NO_CLASSES: MonitoringClass[] = [];

// What the wizard page needs from the Monitoring step: whether to show it,
// how to validate its entries and how to put them into the create request.
export const useMonitoringStep = (namespace: string, provider: string) => {
  const { data: classes = NO_CLASSES } =
    useMonitoringClassesForProvider(provider);
  const instanceForms = useMonitoringInstanceForms(namespace, provider);

  const parametersSchema = useMemo(
    () => buildMonitoringParametersSchema(instanceForms),
    [instanceForms]
  );
  const writeEntries = useCallback(
    (formData: Record<string, unknown>) =>
      writeMonitoringEntries(formData, instanceForms),
    [instanceForms]
  );

  return {
    hasMonitoringStep: classes.length > 0,
    parametersSchema,
    writeEntries,
  };
};
