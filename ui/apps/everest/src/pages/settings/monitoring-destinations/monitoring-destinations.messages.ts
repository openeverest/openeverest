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
import { DestinationUsage } from './monitoring-destinations.types';

const databases = (count: number) =>
  count === 1 ? '1 database' : `${count} databases`;

export const Messages = {
  add: 'Add destination',
  noData: 'No monitoring destinations yet',
  columns: {
    name: 'Name',
    namespace: 'Namespace',
    type: 'Type',
    address: 'Address',
    status: 'Status',
    usedBy: 'Used by',
  },
  none: '—',
  state: {
    [MonitoringDestinationState.Ready]: 'Ready',
    [MonitoringDestinationState.Pending]: 'Pending',
    [MonitoringDestinationState.Failing]: 'Not ready',
    [MonitoringDestinationState.Deleting]: 'Deleting',
  } satisfies Record<MonitoringDestinationState, string>,
  serverVersion: (version: string) => `v${version}`,
  usedByCount: databases,
  usedByDialogTitle: (name: string) => `Databases using ${name}`,
  hiddenInstances: (count: number) =>
    `${databases(count)} you don't have access to`,
  edit: 'Edit',
  delete: 'Delete',
  cancel: 'Cancel',
  close: 'Close',
  deleteHeader: 'Delete monitoring destination',
  deleteConfirmation: (name: string) =>
    `Are you sure you want to delete the ${name} destination?`,
  secretDeletedToo: (secret: string) =>
    `Its credentials Secret ${secret} is deleted with it.`,
  inUseHeader: 'Destination in use',
  inUse: (
    name: string,
    { count, instances }: DestinationUsage,
    shownLimit: number
  ) => {
    const shown = instances.slice(0, shownLimit);
    if (shown.length === 0) {
      return `${name} is used by ${databases(count)}. Remove it from them first.`;
    }
    const more = count - shown.length;
    const list =
      more > 0 ? `${shown.join(', ')} and ${more} more` : shown.join(', ');
    return `${name} is used by ${list}. Remove it from those databases first.`;
  },
};
