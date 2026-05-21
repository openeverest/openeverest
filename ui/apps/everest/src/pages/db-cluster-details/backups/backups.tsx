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

import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages.ts';
import { useContext, useState } from 'react';
import { useParams } from 'react-router-dom';
import { BackupsList } from './backups-list/backups-list';
import { ScheduleModalContext } from './backups.context.ts';
import { NoStoragesMessage } from './no-storages-message/no-storages-message';
import { OnDemandBackupModal } from './on-demand-backup-modal/on-demand-backup-modal';
import { ScheduledBackupModal } from './scheduled-backup-modal';
import { ScheduleWizardMode, WizardMode } from 'shared-types/wizard.types.ts';
import { DbInstanceContext } from '../dbCluster.context';

export const Backups = () => {
  const { namespace = '' } = useParams();
  const { instance, isLoading: instanceLoading } =
    useContext(DbInstanceContext);

  const { data: backupStorages = [], isLoading } =
    useBackupStoragesByNamespace(namespace);

  const [mode, setMode] = useState<ScheduleWizardMode>(WizardMode.New);
  const [openScheduleModal, setOpenScheduleModal] = useState(false);
  const [openOnDemandModal, setOpenOnDemandModal] = useState(false);
  const [selectedScheduleName, setSelectedScheduleName] = useState<string>('');

  const noStorages = !backupStorages.length;

  if (!instance || isLoading || instanceLoading) {
    return null;
  }

  return (
    <ScheduleModalContext.Provider
      value={{
        instance,
        mode,
        setMode,
        openScheduleModal,
        setOpenScheduleModal,
        selectedScheduleName,
        setSelectedScheduleName,
        openOnDemandModal,
        setOpenOnDemandModal,
      }}
    >
      {noStorages ? (
        <NoStoragesMessage />
      ) : (
        <>
          <BackupsList />
          {openOnDemandModal && <OnDemandBackupModal />}
          {openScheduleModal && <ScheduledBackupModal />}
        </>
      )}
    </ScheduleModalContext.Provider>
  );
};
