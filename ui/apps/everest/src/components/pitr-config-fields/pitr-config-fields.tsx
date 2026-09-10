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

import { useBackupClassUiSchema } from 'hooks/api/backup-classes/useBackupClasses';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { useApplySchemaDefaults } from 'components/ui-generator/hooks/use-apply-schema-defaults';
import { PitrConfigFieldsProps } from './pitr-config-fields.types';

export const PitrConfigFields = ({
  backupClass,
  namespace,
}: PitrConfigFieldsProps) => {
  const { sections } = useBackupClassUiSchema(backupClass);
  const className = backupClass?.metadata?.name ?? '';

  // Apply provider defaults only where the form has no value yet, so
  // previously saved parameters are preserved.
  useApplySchemaDefaults(sections?.pitr?.components, className, {
    onlyWhenEmpty: true,
  });

  if (!sections?.pitr) {
    return null;
  }

  return (
    <UIGenerator
      sectionKey="pitr"
      sections={sections}
      formMode={FormMode.Edit}
      namespace={namespace}
    />
  );
};
