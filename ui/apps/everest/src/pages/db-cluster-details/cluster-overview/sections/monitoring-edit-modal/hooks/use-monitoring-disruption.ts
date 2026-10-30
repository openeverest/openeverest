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
import { useFormContext, useWatch } from 'react-hook-form';
import { useMonitoringClassesForProvider } from 'hooks/api/monitoring-classes';
import { useMonitoringDestinations } from 'hooks/api/monitoring-destinations';
import {
  MONITORING_DESTINATIONS_FIELD,
  parseMonitoringDestinations,
} from 'pages/database-form/database-form-body/steps/monitoring-step';
import { Instance } from 'shared-types/api.types';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { getInstanceMonitoringDestinationNames } from 'utils/monitoring-destinations';
import { MonitoringDisruption } from '../monitoring-edit-modal.types';
import { getMonitoringDisruption } from '../monitoring-edit-modal.utils';

const NO_CLASSES: MonitoringClass[] = [];
const NO_DESTINATIONS: MonitoringDestination[] = [];

// The disruption the destinations picked in the form cause, compared with
// those the Instance has now.
export const useMonitoringDisruption = (
  instance: Instance,
  namespace: string
): MonitoringDisruption => {
  const provider = instance.spec?.providerRef.name ?? '';
  const { control } = useFormContext();
  const entries = useWatch({ control, name: MONITORING_DESTINATIONS_FIELD });
  const { data: classes = NO_CLASSES } =
    useMonitoringClassesForProvider(provider);
  const { data: destinations = NO_DESTINATIONS } =
    useMonitoringDestinations(namespace);

  return useMemo(
    () =>
      getMonitoringDisruption({
        before: getInstanceMonitoringDestinationNames(instance),
        after: parseMonitoringDestinations(entries).map(
          (entry) => entry.destinationRef.name
        ),
        destinations,
        classes,
        provider,
        engineReplicas: instance.spec?.components?.engine?.replicas,
      }),
    [instance, entries, destinations, classes, provider]
  );
};
