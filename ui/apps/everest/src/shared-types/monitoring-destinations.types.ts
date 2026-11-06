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

import { CrdsGen } from '@generated/api-types';
import { Instance } from './api.types';

export type MonitoringClass = CrdsGen.components['schemas']['MonitoringClass'];
export type MonitoringClassList =
  CrdsGen.components['schemas']['MonitoringClassList'];
export type MonitoringDestination =
  CrdsGen.components['schemas']['MonitoringDestination'];
export type MonitoringDestinationList =
  CrdsGen.components['schemas']['MonitoringDestinationList'];
export type MonitoringBinding =
  CrdsGen.components['schemas']['MonitoringBinding'];
export type MonitoringBindingList =
  CrdsGen.components['schemas']['MonitoringBindingList'];
export type MonitoringAuthMode = MonitoringClass['spec']['authModes'][number];
export type MonitoringDestinationSpec = MonitoringDestination['spec'];

// Condition types core and the class controllers report on MonitoringClass
// and MonitoringDestination; the generated types keep `type` a plain string.
export const MonitoringConditionType = {
  Accepted: 'Accepted',
  Ready: 'Ready',
} as const;

export type InstanceMonitoringSpec = NonNullable<
  NonNullable<Instance['spec']>['monitoring']
>;
export type InstanceMonitoringDestination = NonNullable<
  InstanceMonitoringSpec['destinations']
>[number];

// The generated types keep `parameters` opaque; what the UI writes is the
// class-defined object.
export type InstanceMonitoringDestinationPayload = Omit<
  InstanceMonitoringDestination,
  'parameters'
> & { parameters?: Record<string, unknown> };

export type MonitoringDestinationPayload = Omit<
  MonitoringDestination,
  'spec' | 'status'
> & {
  spec: Omit<MonitoringDestinationSpec, 'parameters'> & {
    parameters?: Record<string, unknown>;
  };
};

export enum MonitoringDestinationState {
  Ready = 'ready',
  Pending = 'pending',
  Failing = 'failing',
}

// Accepted (core) and Ready (class controller) rolled into one state, with
// the condition that explains it.
export interface MonitoringDestinationStatus {
  state: MonitoringDestinationState;
  reason?: string;
  message?: string;
  serverVersion?: string;
}
