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

import { useMemo } from 'react';
import { FormDialog } from 'components/form-dialog';
import { removeEmptyFieldValues } from 'components/ui-generator/utils/postprocess/postprocess-schema';
import { useBackupClassUiSchema } from 'hooks/api/backup-classes/useBackupClasses';
import { PitrConfigFields } from 'components/pitr-config-fields';
import { pitrConfigSchema } from './pitr-config-modal-schema';
import { PitrConfigModalProps } from './pitr-config-modal.types';
import { Messages } from './pitr-config-modal.messages';

// UIGenerator binds each `pitr` field under `parameters.<leaf>` (the schema's
// relative path), so form data nests as { parameters: { ... } }.
type PitrFormData = { parameters?: Record<string, unknown> } & Record<
  string,
  unknown
>;

export const PitrConfigModal = ({
  open,
  storageName,
  backupClass,
  currentParameters,
  submitting,
  namespace,
  onClose,
  onSubmit,
}: PitrConfigModalProps) => {
  const { sections } = useBackupClassUiSchema(backupClass);
  const schema = useMemo(() => pitrConfigSchema(sections), [sections]);
  const defaultValues = useMemo<PitrFormData>(
    () => ({ parameters: currentParameters ?? {} }),
    [currentParameters]
  );

  const handleSubmit = (data: PitrFormData) => {
    const cleaned = removeEmptyFieldValues(data.parameters ?? {});
    onSubmit(Object.keys(cleaned).length > 0 ? cleaned : undefined);
  };

  return (
    <FormDialog<PitrFormData>
      isOpen={open}
      closeModal={onClose}
      headerMessage={Messages.title(storageName)}
      onSubmit={handleSubmit}
      submitting={submitting}
      submitMessage={Messages.save}
      schema={schema}
      defaultValues={defaultValues}
    >
      <PitrConfigFields backupClass={backupClass} namespace={namespace} />
    </FormDialog>
  );
};
