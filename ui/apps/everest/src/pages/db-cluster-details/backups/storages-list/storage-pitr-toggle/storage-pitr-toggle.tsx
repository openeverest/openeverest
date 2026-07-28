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

import { FormControlLabel, Switch, Tooltip } from '@mui/material';
import { InstanceBackupStorage } from '../../backups.types';
import { useStoragePitr } from './use-storage-pitr';
import { Messages } from './storage-pitr-toggle.messages';

interface StoragePitrToggleProps {
  storage: InstanceBackupStorage;
}

export const StoragePitrToggle = ({ storage }: StoragePitrToggleProps) => {
  const { visible, enabled, disabled, reason, setEnabled } =
    useStoragePitr(storage);

  if (!visible) {
    return null;
  }

  const toggle = (
    <FormControlLabel
      label={Messages.pitr}
      labelPlacement="start"
      control={
        <Switch
          size="small"
          checked={enabled}
          disabled={disabled}
          onChange={(_, checked) => setEnabled(checked)}
          data-testid={`pitr-toggle-${storage.storageRef.name}`}
        />
      }
    />
  );

  // A disabled MUI control swallows hover events, so wrap it in a span so the
  // tooltip explaining why PITR is unavailable still shows.
  return reason ? (
    <Tooltip title={reason}>
      <span>{toggle}</span>
    </Tooltip>
  ) : (
    toggle
  );
};
