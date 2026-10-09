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

import { useMemo, useState } from 'react';
import { FormDialog } from 'components/form-dialog';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { getDefaultValues } from 'components/ui-generator/utils/default-values';
import { postprocessSchemaData } from 'components/ui-generator/utils/postprocess/postprocess-schema';
import { useCreateSecret } from 'hooks/api/secrets';
import { useClusterName } from 'hooks/api/useClusterName';
import { SecretNameInput } from './secret-name-input';
import { secretCreateSchema } from './secret-create-modal-schema';
import {
  buildSecretPayload,
  generateSecretName,
  toSecretSchema,
} from './secret-create-modal.utils';
import {
  SECRET_DATA_ROOT,
  SECRET_SECTION_KEY,
} from './secret-create-modal.constants';
import {
  SecretCreateModalProps,
  SecretFormData,
} from './secret-create-modal.types';
import { Messages } from './secret-create-modal.messages';

// Renders the definition's create form at the secret's data and creates the secret on submit.
export const SecretCreateModal = ({
  definition,
  section,
  providerObject,
  namespace,
  generator: Generator,
  onClose,
  onCreated,
}: SecretCreateModalProps) => {
  const cluster = useClusterName();
  const { mutate, isPending } = useCreateSecret(cluster, namespace);
  const [takenNames, setTakenNames] = useState<string[]>([]);

  const schema = useMemo(
    () => toSecretSchema(section, providerObject),
    [section, providerObject]
  );
  const sections = schema[SECRET_SECTION_KEY].sections;

  const { schema: formSchema, celDependencyGroups } = useMemo(
    () => secretCreateSchema(sections, takenNames),
    [sections, takenNames]
  );

  const defaultValues = useMemo<SecretFormData>(
    () => ({
      metadata: { name: generateSecretName(definition) },
      [SECRET_DATA_ROOT]: getDefaultValues(schema, SECRET_SECTION_KEY),
    }),
    [schema, definition]
  );

  const handleSubmit = (data: SecretFormData) => {
    const { name } = data.metadata;
    const values = postprocessSchemaData(data[SECRET_DATA_ROOT], {
      schema,
      selectedTopology: SECRET_SECTION_KEY,
    });

    mutate(
      buildSecretPayload({
        name,
        definition,
        provider: providerObject.metadata?.name ?? '',
        values,
      }),
      {
        onSuccess: () => onCreated(name),
        onError: (error) => {
          if (error.status === 409) setTakenNames((names) => [...names, name]);
        },
      }
    );
  };

  return (
    <FormDialog<SecretFormData>
      isOpen
      closeModal={onClose}
      headerMessage={section.label ?? Messages.title}
      description={section.description}
      schema={formSchema}
      celDependencyGroups={celDependencyGroups}
      defaultValues={defaultValues}
      onSubmit={handleSubmit}
      submitting={isPending}
      submitMessage={Messages.create}
      size="XL"
    >
      <SecretNameInput takenNames={takenNames} />
      <Generator
        sectionKey={SECRET_SECTION_KEY}
        sections={sections}
        root={SECRET_DATA_ROOT}
        providerObject={providerObject}
        namespace={namespace}
        formMode={FormMode.New}
      />
    </FormDialog>
  );
};
