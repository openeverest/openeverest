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
import { useMonitoringClassesForProvider } from 'hooks/api/monitoring-classes';
import { useMonitoringDestinations } from 'hooks/api/monitoring-destinations';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { MonitoringSectionForm } from 'utils/monitoring-classes';
import { buildInstanceFormsByDestination } from '../monitoring-step.utils';

const NO_CLASSES: MonitoringClass[] = [];
const NO_DESTINATIONS: MonitoringDestination[] = [];

// Keyed by destination name: the form that fills a monitoring entry's
// parameters, for the destinations whose class defines one.
export const useMonitoringInstanceForms = (
  namespace: string,
  provider: string
): Map<string, MonitoringSectionForm> => {
  const { data: classes = NO_CLASSES } =
    useMonitoringClassesForProvider(provider);
  const { data: destinations = NO_DESTINATIONS } =
    useMonitoringDestinations(namespace);

  return useMemo(
    () => buildInstanceFormsByDestination(classes, destinations),
    [classes, destinations]
  );
};
