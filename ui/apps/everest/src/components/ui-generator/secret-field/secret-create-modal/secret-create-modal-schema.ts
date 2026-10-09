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

import { z } from 'zod';
import { FormMode, Section } from 'components/ui-generator/ui-generator.types';
import { buildSectionZodSchema } from 'components/ui-generator/utils/schema-builder';
import { joinPath } from 'components/ui-generator/utils/object-path';
import { rfc_123_schema } from 'utils/common-validation';
import {
  MAX_SECRET_NAME_LENGTH,
  SECRET_DATA_ROOT,
  SECRET_SECTION_KEY,
} from './secret-create-modal.constants';
import { Messages } from './secret-create-modal.messages';

// `takenNames` holds names the server rejected as already in use.
export const secretCreateSchema = (
  sections: Record<string, Section>,
  takenNames: string[]
) => {
  const { schema, celDependencyGroups } = buildSectionZodSchema(
    SECRET_SECTION_KEY,
    sections,
    { formMode: FormMode.New }
  );

  return {
    schema: z.object({
      metadata: z.object({
        name: rfc_123_schema({
          fieldName: Messages.nameFieldName,
          maxLength: MAX_SECRET_NAME_LENGTH,
        }).refine((name) => !takenNames.includes(name), Messages.nameTaken),
      }),
      [SECRET_DATA_ROOT]: schema,
    }),
    celDependencyGroups: celDependencyGroups.map((group) =>
      group.map((path) => joinPath(SECRET_DATA_ROOT, path))
    ),
  };
};
