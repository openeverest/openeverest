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

import { GetRestorePayload, Restore } from 'shared-types/restores.types';

type RestoreListItem = NonNullable<GetRestorePayload['items']>[number];

// Flatten a Restore CRD list item into the table view model. A PointInTime
// restore surfaces its source storage; a backup restore surfaces its backup.
export const mapRestore = (item: RestoreListItem): Restore => ({
  name: item.metadata?.name ?? '',
  startTime: item.status?.startedAt || item.metadata?.creationTimestamp || '',
  endTime: item.status?.completedAt,
  state: item.status?.state || 'unknown',
  type: item.spec.dataSource.type === 'PointInTime' ? 'pitr' : 'full',
  backupSource:
    item.spec.dataSource.backup?.backupRef.name ??
    item.spec.dataSource.pointInTime?.source.storageRef.name ??
    '',
});
