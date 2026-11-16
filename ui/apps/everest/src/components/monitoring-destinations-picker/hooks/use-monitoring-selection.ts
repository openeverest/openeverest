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
import { useController, useFormContext } from 'react-hook-form';
import { getByPath } from 'components/ui-generator/utils/object-path';
import { useMonitoringClassesForProvider } from 'hooks/api/monitoring-classes';
import { useMonitoringDestinations } from 'hooks/api/monitoring-destinations';
import { useRBACPermissions } from 'hooks/rbac';
import {
  InstanceMonitoringDestination,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { getMonitoringClassName } from 'utils/monitoring-classes';
import { getDestinationName } from 'utils/monitoring-destinations';
import {
  MAX_MONITORING_DESTINATIONS,
  MONITORING_DESTINATIONS_FIELD,
} from '../monitoring-destinations-picker.constants';
import {
  DestinationDisabledReason,
  MonitoringSelection,
  MonitoringSelectionOptions,
} from '../monitoring-destinations-picker.types';
import {
  getDefaultEntryParameters,
  getDestinationDisabledReason,
  getDestinationsUnavailableReason,
  getUnavailableEntries,
  groupDestinationsByClass,
  isDestinationSelected,
  parseMonitoringDestinations,
  pruneMonitoringDestinations,
  toggleMonitoringDestination,
  toggleMonitoringEntry,
} from '../monitoring-destinations-picker.utils';

// Works on the spec.monitoring.destinations field of the surrounding form.
export const useMonitoringSelection = ({
  namespace: namespaceName,
  provider: providerName,
  pruneUnavailable,
  instanceForms,
}: MonitoringSelectionOptions): MonitoringSelection => {
  const { control, getValues, formState } = useFormContext();

  const {
    data: classes = [],
    isLoading: classesLoading,
    isError: classesFailed,
  } = useMonitoringClassesForProvider(providerName);
  const {
    data: namespaceDestinations = [],
    isLoading: destinationsLoading,
    isSuccess: destinationsLoaded,
    isError: destinationsFailed,
  } = useMonitoringDestinations(namespaceName);
  const { canRead: canReadDestinations, isPermissionsLoading } =
    useRBACPermissions('monitoring-destinations', `${namespaceName}/*`);

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

  // The entries the form started from: an Instance's own, which it keeps.
  const initialEntries = useMemo(
    () =>
      parseMonitoringDestinations(
        getByPath(formState.defaultValues ?? {}, MONITORING_DESTINATIONS_FIELD)
      ),
    [formState.defaultValues]
  );
  const unavailableEntries = useMemo(
    () =>
      pruneUnavailable
        ? []
        : getUnavailableEntries(
            initialEntries,
            namespaceDestinations,
            destinations
          ).map(({ entry, reason }) => {
            const selectedEntry = isDestinationSelected(
              selected,
              entry.destinationRef.name
            );
            return {
              entry,
              reason,
              selected: selectedEntry,
              disabledReason:
                !selectedEntry && selected.length >= MAX_MONITORING_DESTINATIONS
                  ? DestinationDisabledReason.LimitReached
                  : undefined,
            };
          }),
    [
      pruneUnavailable,
      initialEntries,
      namespaceDestinations,
      destinations,
      selected,
    ]
  );

  useEffect(() => {
    if (!pruneUnavailable || !destinationsLoaded || classesLoading) {
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
    pruneUnavailable,
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

  // Removes an entry the Instance had, or puts it back as it was.
  const toggleEntry = useCallback(
    (entry: InstanceMonitoringDestination) =>
      onChange(
        toggleMonitoringEntry(
          parseMonitoringDestinations(getValues(MONITORING_DESTINATIONS_FIELD)),
          entry
        )
      ),
    [getValues, onChange]
  );

  return {
    namespace: namespaceName,
    groups,
    unavailableEntries,
    isLoading: classesLoading || destinationsLoading || isPermissionsLoading,
    unavailableReason: getDestinationsUnavailableReason({
      canRead: canReadDestinations,
      loadFailed: classesFailed || destinationsFailed,
    }),
    isSelected,
    getDisabledReason,
    toggle,
    toggleEntry,
    getEntryIndex,
    getInstanceForm,
  };
};
