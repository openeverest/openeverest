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

import { useContext, useState } from 'react';
import { format } from 'date-fns';
import { DATE_FORMAT } from 'consts';
import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';
import { PitrStorageRow } from 'components/pitr-storage-row';
import { PitrConfigModal } from 'components/pitr-config-modal';
import { ScheduleModalContext } from '../../backups.context';
import { InstanceBackupStorage } from '../../backups.types';
import { getStoragePitrWindow } from '../../pitr.utils';
import { useStoragePitr } from './use-storage-pitr';
import { Messages } from './storage-pitr-toggle.messages';

interface StoragePitrToggleProps {
  storage: InstanceBackupStorage;
}

export const StoragePitrToggle = ({ storage }: StoragePitrToggleProps) => {
  const { instance } = useContext(ScheduleModalContext);
  const {
    visible,
    enabled,
    disabled,
    reason,
    showConfig,
    configDisabled,
    activeClass,
    currentParameters,
    namespace,
    isPending,
    setEnabled,
    setParameters,
  } = useStoragePitr(storage);
  const [confirmingDisable, setConfirmingDisable] = useState(false);
  const [configuring, setConfiguring] = useState(false);

  if (!visible) {
    return null;
  }

  const storageName = storage.storageRef.name;

  // Show the restorable range only once the provider reports a window; no
  // placeholder while it's absent.
  const window = enabled
    ? getStoragePitrWindow(instance, storageName)
    : undefined;
  const caption =
    window?.available && window.earliest && window.latest
      ? Messages.recoveryWindow(
          format(window.earliest, DATE_FORMAT),
          format(window.latest, DATE_FORMAT)
        )
      : undefined;

  // Enabling is a direct action; disabling always asks for confirmation.
  const handleToggle = (checked: boolean) => {
    if (checked) {
      setEnabled(true);
    } else {
      setConfirmingDisable(true);
    }
  };

  return (
    <>
      <PitrStorageRow
        storageName={storageName}
        checked={enabled}
        onToggle={handleToggle}
        toggleDisabled={disabled}
        toggleReason={reason}
        showConfig={showConfig}
        configDisabled={configDisabled}
        onConfigClick={() => setConfiguring(true)}
        caption={caption}
      />

      <ConfirmDialog
        open={confirmingDisable}
        selectedId={storageName}
        closeModal={() => setConfirmingDisable(false)}
        headerMessage={Messages.disable.title}
        cancelMessage={Messages.disable.cancel}
        submitMessage={Messages.disable.confirm}
        handleConfirm={() => {
          setEnabled(false);
          setConfirmingDisable(false);
        }}
      >
        {Messages.disable.body(storageName)}
      </ConfirmDialog>

      {configuring && (
        <PitrConfigModal
          open
          storageName={storageName}
          backupClass={activeClass}
          currentParameters={currentParameters}
          submitting={isPending}
          namespace={namespace}
          onClose={() => setConfiguring(false)}
          onSubmit={(parameters) => {
            setParameters(parameters);
            setConfiguring(false);
          }}
        />
      )}
    </>
  );
};
