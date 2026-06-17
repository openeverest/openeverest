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

import { TimeSelectionFields } from '../../time-selection/time-selection.types';
import { FlattenedSchedule } from '../schedule-form-dialog-context/schedule-form-dialog-context.types';
import { BackupClass } from 'shared-types/backups.types';

enum ScheduleForm {
  scheduleName = 'scheduleName',
  storageLocation = 'storageLocation',
  retentionCopies = 'retentionCopies',
  backupClassName = 'backupClassName',
}

export interface ScheduleFormProps {
  allowScheduleSelection?: boolean;
  disableStorageSelection?: boolean;
  autoFillLocation?: boolean;
  disableNameInput?: boolean;
  schedules: FlattenedSchedule[];
  disableNameEdit?: boolean;
  maxStorages?: number;
  maxSchedulesPerStorage?: number;
  instanceStorageNames?: string[];
  availableClasses: BackupClass[];
  disableClassSelection?: boolean;
  backupClass?: BackupClass;
}

export const ScheduleFormFields = { ...ScheduleForm, ...TimeSelectionFields };
export type ScheduleFormFields = ScheduleForm | TimeSelectionFields;
