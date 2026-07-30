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
import { Switch, Typography } from '@mui/material';
import EditableItem from 'components/editable-item/editable-item';
import { BlockedTooltip } from 'components/blocked-tooltip';
import { BackupClass } from 'shared-types/backups.types';
import { PitrConfigModal } from 'components/pitr-config-modal';
import { WizardPitrStorage } from './use-pitr-block';
import { Messages } from './pitr-block.messages';

interface PitrStorageRowProps {
  storage: WizardPitrStorage;
  backupClass: BackupClass | undefined;
  namespace: string;
  onToggle: (checked: boolean) => void;
  onSetParameters: (parameters: Record<string, unknown> | undefined) => void;
}

export const PitrStorageRow = ({
  storage,
  backupClass,
  namespace,
  onToggle,
  onSetParameters,
}: PitrStorageRowProps) => {
  const [configuring, setConfiguring] = useState(false);
  const {
    name,
    enabled,
    reason,
    showConfig,
    configDisabled,
    scheduleCount,
    currentParameters,
  } = storage;

  return (
    <>
      <EditableItem
        dataTestId={name}
        endText={Messages.scheduleCount(scheduleCount)}
        editButtonProps={
          showConfig
            ? { onClick: () => setConfiguring(true), disabled: configDisabled }
            : undefined
        }
        controls={
          <BlockedTooltip reason={reason}>
            <Switch
              size="small"
              checked={enabled}
              disabled={reason !== undefined}
              onChange={(_, next) => onToggle(next)}
              data-testid={`pitr-toggle-${name}`}
            />
          </BlockedTooltip>
        }
      >
        <Typography variant="body1">{name}</Typography>
      </EditableItem>

      {configuring && (
        <PitrConfigModal
          open
          storageName={name}
          backupClass={backupClass}
          currentParameters={currentParameters}
          namespace={namespace}
          onClose={() => setConfiguring(false)}
          onSubmit={(parameters) => {
            onSetParameters(parameters);
            setConfiguring(false);
          }}
        />
      )}
    </>
  );
};
