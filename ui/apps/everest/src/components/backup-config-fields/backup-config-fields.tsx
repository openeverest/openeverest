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

import { useEffect, useRef } from 'react';
import { useFormContext } from 'react-hook-form';
import { useBackupClassUiSchema } from 'hooks/api/backup-classes/useBackupClasses';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { buildDefaultsFromComponents } from 'components/ui-generator/utils/default-values/build-defaults-from-components';
import { BackupConfigFieldsProps } from './backup-config-fields.types';

export const BackupConfigFields = ({
  backupClass,
  formMode,
  namespace,
}: BackupConfigFieldsProps) => {
  const { setValue, trigger } = useFormContext();
  const appliedDefaultsClassRef = useRef<string>('');

  const { sections: backupSections } = useBackupClassUiSchema(backupClass);

  const className = backupClass?.metadata?.name ?? '';

  useEffect(() => {
    if (!className || appliedDefaultsClassRef.current === className) {
      return;
    }

    if (!backupSections) return;

    const explicitDefaults = backupSections.config?.components
      ? buildDefaultsFromComponents(backupSections.config.components, '', true)
      : {};

    Object.entries(explicitDefaults).forEach(([fieldName, defaultValue]) => {
      setValue(fieldName, defaultValue, {
        shouldDirty: false,
        shouldTouch: false,
        shouldValidate: false,
      });
    });

    appliedDefaultsClassRef.current = className;
    trigger();
  }, [backupSections, className, setValue, trigger]);

  if (!backupSections) {
    return null;
  }

  return (
    <UIGenerator
      sectionKey="config"
      sections={backupSections}
      formMode={formMode}
      namespace={namespace}
    />
  );
};
