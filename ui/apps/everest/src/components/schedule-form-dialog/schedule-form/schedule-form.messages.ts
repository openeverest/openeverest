// everest
// Copyright (C) 2023 Percona LLC
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

import { errorMessages } from 'utils/common-validation';

export const Messages = {
  scheduleName: {
    label: 'Backup name',
    tooLong: errorMessages.tooLong('schedule'),
    duplicate: 'You already have a schedule with the same name.',
  },
  storageLocation: {
    label: 'Backup storage',
    invalidOption:
      'Invalid option. Please make sure you added a backup storage and select it from the dropdown',
  },
  retentionCopies: {
    label: 'Retention copies',
    helperText: '0 = infinite retention copies',
    invalidNumber: 'Invalid number',
  },
  backupDetails: 'Backup Details',
  repeats: 'Repeats',
  backupClass: {
    label: 'Backup class',
    required: 'Please select a backup class',
    disabledHelperText:
      'Backup class is shared across all schedules for this instance.',
  },
  sameTimeSchedule:
    'A schedule with the same time selection already exists. Set a different time to create a new one.',
  sameStorageScheduleForPG:
    'A schedule with the same storage already exists. Set a different storage to create a new one.',
  pgStorageEditRestriction:
    'You cannot modify the storage location for an existing schedule due to a limitation of the PG operator.',
};
