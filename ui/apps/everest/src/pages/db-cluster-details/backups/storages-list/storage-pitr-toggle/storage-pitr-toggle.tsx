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

import { useState } from 'react';
import { Box } from '@mui/material';
import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';
import { PitrToggleSwitch } from 'components/pitr-toggle-switch';
import { PitrConfigButton } from 'components/pitr-config-button';
import { PitrConfigModal } from 'components/pitr-config-modal';
import { InstanceBackupStorage } from '../../backups.types';
import { useStoragePitr } from './use-storage-pitr';
import { Messages } from './storage-pitr-toggle.messages';

interface StoragePitrToggleProps {
  storage: InstanceBackupStorage;
}

export const StoragePitrToggle = ({ storage }: StoragePitrToggleProps) => {
  const {
    visible,
    enabled,
    disabled,
    reason,
    showConfig,
    configDisabled,
    configReason,
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

  // Enabling is a direct action; disabling always asks for confirmation.
  const handleToggle = (checked: boolean) => {
    if (checked) {
      setEnabled(true);
    } else {
      setConfirmingDisable(true);
    }
  };

  return (
    <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 1.5 }}>
      <PitrToggleSwitch
        storageName={storageName}
        checked={enabled}
        disabled={disabled}
        reason={reason}
        onToggle={handleToggle}
      />

      {showConfig && (
        <PitrConfigButton
          storageName={storageName}
          disabled={configDisabled}
          reason={configReason}
          onClick={() => setConfiguring(true)}
        />
      )}

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
    </Box>
  );
};
