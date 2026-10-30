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

export type InstanceMonitoringSpec = NonNullable<
  NonNullable<Instance['spec']>['monitoring']
>;
export type InstanceMonitoringDestination = NonNullable<
  InstanceMonitoringSpec['destinations']
>[number];
