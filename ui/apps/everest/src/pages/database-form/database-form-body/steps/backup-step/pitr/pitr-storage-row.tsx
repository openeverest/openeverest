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
import { BackupClass } from 'shared-types/backups.types';
import { PitrConfigModal } from 'components/pitr-config-modal';
import { PitrStorageRow } from 'components/pitr-storage-row';
import { WizardPitrStorage } from './use-pitr-block';

interface WizardPitrStorageRowProps {
  storage: WizardPitrStorage;
  backupClass: BackupClass | undefined;
  namespace: string;
  onToggle: (checked: boolean) => void;
  onSetParameters: (parameters: Record<string, unknown> | undefined) => void;
}

export const WizardPitrStorageRow = ({
  storage,
  backupClass,
  namespace,
  onToggle,
  onSetParameters,
}: WizardPitrStorageRowProps) => {
  const [configuring, setConfiguring] = useState(false);
  const {
    name,
    enabled,
    reason,
    showConfig,
    configDisabled,
    currentParameters,
  } = storage;

  return (
    <>
      <PitrStorageRow
        storageName={name}
        checked={enabled}
        onToggle={onToggle}
        toggleDisabled={reason !== undefined}
        toggleReason={reason}
        showConfig={showConfig}
        configDisabled={configDisabled}
        onConfigClick={() => setConfiguring(true)}
      />

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
