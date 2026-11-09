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

import { MonitoringDestinationState } from 'shared-types/monitoring-destinations.types';
import { MAX_MONITORING_DESTINATIONS } from './monitoring-destinations-picker.constants';
import {
  DestinationDisabledReason,
  DestinationsUnavailableReason,
} from './monitoring-destinations-picker.types';

export const Messages = {
  noDestinations: (namespace: string) =>
    `No destinations of this system in the ${namespace} namespace yet.`,
  destinationsUnavailable: {
    [DestinationsUnavailableReason.LoadFailed]: () =>
      'Monitoring destinations could not be loaded. Try again later.',
    [DestinationsUnavailableReason.NoAccess]: (namespace: string) =>
      `You don't have permission to view monitoring destinations in the ${namespace} namespace.`,
  } satisfies Record<
    DestinationsUnavailableReason,
    (namespace: string) => string
  >,
  addDestination: 'Add destination',
  state: {
    [MonitoringDestinationState.Ready]: 'Ready',
    [MonitoringDestinationState.Pending]: 'Waiting for its controller',
    [MonitoringDestinationState.Failing]: 'Not ready',
    [MonitoringDestinationState.Deleting]: 'Being deleted',
  } satisfies Record<MonitoringDestinationState, string>,
  disabledReason: {
    [DestinationDisabledReason.LimitReached]: `An instance can send to at most ${MAX_MONITORING_DESTINATIONS} destinations`,
    [DestinationDisabledReason.IntegrationTaken]:
      'Another destination of this system is already selected',
    [DestinationDisabledReason.Deleting]: 'Being deleted',
  } satisfies Record<DestinationDisabledReason, string>,
  entryParameters: (destinationName: string) =>
    `Settings for ${destinationName}`,
};
