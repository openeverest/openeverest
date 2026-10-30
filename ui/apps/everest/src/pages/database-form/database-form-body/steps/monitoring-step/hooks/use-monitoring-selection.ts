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

import { useCallback, useEffect, useMemo } from 'react';
import { useController, useFormContext, useWatch } from 'react-hook-form';
import { DbWizardFormFields } from 'consts';
import { useMonitoringClassesForProvider } from 'hooks/api/monitoring-classes';
import { useMonitoringDestinations } from 'hooks/api/monitoring-destinations';
import { MonitoringDestination } from 'shared-types/monitoring-destinations.types';
import { getMonitoringClassName } from 'utils/monitoring-classes';
import { getDestinationName } from 'utils/monitoring-destinations';
import { MONITORING_DESTINATIONS_FIELD } from '../monitoring-step.constants';
import { MonitoringSelection } from '../monitoring-step.types';
import {
  getDefaultEntryParameters,
  getDestinationDisabledReason,
  groupDestinationsByClass,
  isDestinationSelected,
  parseMonitoringDestinations,
  pruneMonitoringDestinations,
  toggleMonitoringDestination,
} from '../monitoring-step.utils';
import { useMonitoringInstanceForms } from './use-monitoring-instance-forms';

export const useMonitoringSelection = (): MonitoringSelection => {
  const { control, getValues } = useFormContext();
  const namespace = useWatch({
    control,
    name: DbWizardFormFields.k8sNamespace,
  });
  const provider = useWatch({ control, name: DbWizardFormFields.provider });
  const namespaceName = typeof namespace === 'string' ? namespace : '';
  const providerName = typeof provider === 'string' ? provider : '';

  const { data: classes = [], isLoading: classesLoading } =
    useMonitoringClassesForProvider(providerName);
  const {
    data: namespaceDestinations = [],
    isLoading: destinationsLoading,
    isSuccess: destinationsLoaded,
  } = useMonitoringDestinations(namespaceName);
  const instanceForms = useMonitoringInstanceForms(namespaceName, providerName);

  const {
    field: { value, onChange },
  } = useController({
    control,
    name: MONITORING_DESTINATIONS_FIELD,
    defaultValue: [],
  });
  const selected = useMemo(() => parseMonitoringDestinations(value), [value]);

  const classNames = useMemo(
    () => classes.map(getMonitoringClassName),
    [classes]
  );
  const destinations = useMemo(
    () =>
      namespaceDestinations.filter((destination) =>
        classNames.includes(destination.spec.classRef.name)
      ),
    [namespaceDestinations, classNames]
  );
  const groups = useMemo(
    () => groupDestinationsByClass(classes, destinations),
    [classes, destinations]
  );

  useEffect(() => {
    if (!destinationsLoaded || classesLoading) {
      return;
    }
    const current = parseMonitoringDestinations(
      getValues(MONITORING_DESTINATIONS_FIELD)
    );
    const pruned = pruneMonitoringDestinations(
      current,
      destinations.map(getDestinationName)
    );
    if (pruned.length !== current.length) {
      onChange(pruned);
    }
  }, [
    destinationsLoaded,
    classesLoading,
    destinations,
    selected,
    getValues,
    onChange,
  ]);

  const isSelected = useCallback(
    (destination: MonitoringDestination) =>
      isDestinationSelected(selected, getDestinationName(destination)),
    [selected]
  );

  const getDisabledReason = useCallback(
    (destination: MonitoringDestination) =>
      getDestinationDisabledReason({
        destination,
        selected,
        destinations,
        classes,
        provider: providerName,
      }),
    [selected, destinations, classes, providerName]
  );

  const getInstanceForm = useCallback(
    (destination: MonitoringDestination) =>
      instanceForms.get(getDestinationName(destination)),
    [instanceForms]
  );

  const getEntryIndex = useCallback(
    (destination: MonitoringDestination) =>
      selected.findIndex(
        (entry) => entry.destinationRef.name === getDestinationName(destination)
      ),
    [selected]
  );

  // Reads the entries fresh: the controller value does not follow edits of the
  // entries' parameters, and toggling must keep them.
  const toggle = useCallback(
    (destination: MonitoringDestination) =>
      onChange(
        toggleMonitoringDestination(
          parseMonitoringDestinations(getValues(MONITORING_DESTINATIONS_FIELD)),
          getDestinationName(destination),
          getDefaultEntryParameters(getInstanceForm(destination))
        )
      ),
    [getValues, onChange, getInstanceForm]
  );

  return {
    namespace: namespaceName,
    groups,
    isLoading: classesLoading || destinationsLoading,
    isSelected,
    getDisabledReason,
    toggle,
    getEntryIndex,
    getInstanceForm,
  };
};
