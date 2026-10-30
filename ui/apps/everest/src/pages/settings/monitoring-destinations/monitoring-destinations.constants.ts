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

import { BaseStatus } from 'components/status-field/status-field.types';
import { MonitoringDestinationState } from 'shared-types/monitoring-destinations.types';

export const DESTINATION_STATE_TO_STATUS: Record<
  MonitoringDestinationState,
  BaseStatus
> = {
  [MonitoringDestinationState.Ready]: 'success',
  [MonitoringDestinationState.Pending]: 'pending',
  [MonitoringDestinationState.Failing]: 'error',
};

// Database names spelled out in the "in use" message before "and N more".
export const IN_USE_NAMES_SHOWN = 5;
