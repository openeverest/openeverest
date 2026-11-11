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
  MonitoringDestination,
  MonitoringDestinationStatus,
} from 'shared-types/monitoring-destinations.types';

// Core's count, which includes Instances the user can't read, and the names
// the user can read.
export interface DestinationUsage {
  count: number;
  instances: string[];
}

export interface MonitoringDestinationRow {
  name: string;
  namespace: string;
  type: string;
  address?: string;
  status: MonitoringDestinationStatus;
  usedBy: DestinationUsage;
  raw: MonitoringDestination;
}

// A destination the user asked to delete. inUse is set when the server
// refuses to delete it.
export interface MonitoringDestinationDeletion {
  row: MonitoringDestinationRow;
  inUse?: DestinationUsage;
}
