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
import { useProviders } from 'hooks/api/providers/useProviders';
import {
  finalizeMonitoringEntries,
  MONITORING_DESTINATIONS_FIELD,
  parseMonitoringDestinations,
  useMonitoringInstanceForms,
} from 'components/monitoring-destinations-picker';
import { Instance, Provider } from 'shared-types/api.types';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { MonitoringDisruption } from '../monitoring-edit-modal.types';
import {
  getMonitoringDisruption,
  hasSingleReplicaMonitoredComponent,
} from '../monitoring-edit-modal.utils';

const NO_CLASSES: MonitoringClass[] = [];
const NO_DESTINATIONS: MonitoringDestination[] = [];
const NO_PROVIDERS: Provider[] = [];

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
  const { data: providers = NO_PROVIDERS } = useProviders();
  const instanceForms = useMonitoringInstanceForms(namespace, provider);

  return useMemo(
    () =>
      getMonitoringDisruption({
        before: instance.spec?.monitoring?.destinations ?? [],
        // Compared as they would be saved, so an untouched entry is no change.
        after: finalizeMonitoringEntries(
          parseMonitoringDestinations(entries),
          instanceForms
        ),
        destinations,
        classes,
        provider,
        hasSingleReplica: hasSingleReplicaMonitoredComponent(
          instance,
          providers.find((p) => p.metadata?.name === provider)
        ),
      }),
    [
      instance,
      entries,
      instanceForms,
      destinations,
      classes,
      provider,
      providers,
    ]
  );
};
