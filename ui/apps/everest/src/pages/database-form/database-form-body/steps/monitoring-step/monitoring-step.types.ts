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

import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { MonitoringSectionForm } from 'utils/monitoring-classes';

export enum DestinationDisabledReason {
  LimitReached = 'limitReached',
  IntegrationTaken = 'integrationTaken',
}

export interface MonitoringClassDestinations {
  monitoringClass: MonitoringClass;
  destinations: MonitoringDestination[];
}

export interface MonitoringSelection {
  namespace: string;
  groups: MonitoringClassDestinations[];
  isLoading: boolean;
  isSelected: (destination: MonitoringDestination) => boolean;
  getDisabledReason: (
    destination: MonitoringDestination
  ) => DestinationDisabledReason | undefined;
  toggle: (destination: MonitoringDestination) => void;
  // Position of the destination's entry in spec.monitoring.destinations, -1
  // when it is not selected.
  getEntryIndex: (destination: MonitoringDestination) => number;
  getInstanceForm: (
    destination: MonitoringDestination
  ) => MonitoringSectionForm | undefined;
}

export interface MonitoringSelectionOptions {
  namespace: string;
  provider: string;
  // Drop entries whose destination is not offered, e.g. after the namespace
  // changed on an earlier wizard step. An existing Instance keeps them, so an
  // edit never removes what it doesn't show.
  pruneUnavailable: boolean;
}
