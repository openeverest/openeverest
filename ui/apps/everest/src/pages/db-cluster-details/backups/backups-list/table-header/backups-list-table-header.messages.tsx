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

export const Messages = {
  now: 'Now',
  schedule: 'Schedule',
  deleteModal: {
    header: 'Delete schedule',
    content: (scheduleName: string, willDisablePITR: boolean) => (
      <>
        Are you sure you want to permanently delete schedule{' '}
        <b>{scheduleName}</b>?{' '}
        {willDisablePITR &&
          ' This will disable point-in-time recovery, as it requires a full backup.'}
      </>
    ),
  },
  activeSchedules: (schedulesNumber: number) =>
    `${schedulesNumber} active schedule${schedulesNumber > 1 ? 's' : ''}`,
  exceededScheduleBackupsNumber: (maxStorages: number) =>
    `Maximum number of storages (${maxStorages}) for this backup class has been reached.`,
  noStoragesAvailable:
    'Add a new backup storage in order to create a backup schedule',
};
