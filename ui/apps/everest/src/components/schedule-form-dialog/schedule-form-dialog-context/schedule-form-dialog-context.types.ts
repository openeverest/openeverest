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

import { DbCluster } from 'shared-types/dbCluster.types';
import { ScheduleFormData } from '../schedule-form/schedule-form-schema';
import { kebabize } from '@percona/utils';
import { ScheduleWizardMode, WizardMode } from 'shared-types/wizard.types';

export type ScheduleFormDialogExternalContext =
  | 'db-wizard-new'
  | 'db-wizard-edit'
  | 'db-wizard-restore-from-backup'
  | 'db-details-backups';
export const dbWizardToScheduleFormDialogMap = (dbWizardMode: WizardMode) => {
  return `db-wizard-${kebabize(
    dbWizardMode
  )}` as ScheduleFormDialogExternalContext;
};
export type ScheduleFormDialogContextType = {
  mode: ScheduleWizardMode;
  externalContext?: ScheduleFormDialogExternalContext;
  setMode: React.Dispatch<React.SetStateAction<ScheduleWizardMode>>;
  selectedScheduleName: string;
  setSelectedScheduleName: React.Dispatch<React.SetStateAction<string>>;
  openScheduleModal: boolean;
  setOpenScheduleModal: React.Dispatch<React.SetStateAction<boolean>>;
  handleClose: () => void;
  handleSubmit: (data: ScheduleFormData) => void;
  isPending: boolean;
  dbInstanceInfo: {
    dbInstanceName?: DbCluster['metadata']['name'];
    namespace: DbCluster['metadata']['namespace'];
    schedules: NonNullable<DbCluster['spec']['backup']>['schedules'];
    activeStorage?: NonNullable<DbCluster['status']>['activeStorage'];
    dbEngine: DbCluster['spec']['engine']['type'];
    defaultSchedules: NonNullable<DbCluster['spec']['backup']>['schedules'];
  };
};
