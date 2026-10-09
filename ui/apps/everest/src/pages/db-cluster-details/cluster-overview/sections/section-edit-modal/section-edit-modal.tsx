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
import { enqueueSnackbar } from 'notistack';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { buildSectionZodSchema } from 'components/ui-generator/utils/schema-builder';
import { extractInstanceValues } from 'components/ui-generator/utils/default-values/extract-instance-values';
import { applyModeOverrides } from 'components/ui-generator/utils/preprocess/apply-mode-overrides';
import { mergeSectionEdit } from 'components/ui-generator/utils/postprocess/merge-section-edit';
import {
  deepClone,
  deepMerge,
} from 'components/ui-generator/utils/object-path/object-path';
import { useUpdateDbInstanceWithConflictRetry } from 'hooks/api/db-instances/useUpdateDbInstance';
import { useKubernetesClusterInfo } from 'hooks/api/kubernetesClusters/useKubernetesClusterInfo';
import type { Instance } from 'shared-types/api.types';
import { widgetRegistry } from 'pages/database-form/widget-registry';
import type { SectionEditModalProps } from './section-edit-modal.types';
import { applyRuntimeOverrides } from './section-edit-modal.utils';
import { Messages } from './section-edit-modal.messages';

const SectionEditModal = ({
  sectionKey,
  sections,
  instance,
  provider,
  namespace,
  onClose,
  onSuccess,
}: SectionEditModalProps) => {
  const [submitting, setSubmitting] = useState(false);
  const { data: clusterInfo } = useKubernetesClusterInfo([
    'section-edit-cluster-info',
  ]);

  const editSections = useMemo(() => {
    const modeApplied = applyModeOverrides(sections, FormMode.Edit);
    return applyRuntimeOverrides(modeApplied, instance, clusterInfo);
  }, [sections, instance, clusterInfo]);

  const section = editSections[sectionKey];

  const defaultValues = useMemo(
    () =>
      extractInstanceValues(
        editSections,
        instance as unknown as Record<string, unknown>,
        FormMode.Edit
      ),
    [editSections, instance]
  );

  // CEL compares form values with `original`, so both must be read the same way ("10Gi" → 10).
  const originalDataForCel = useMemo(
    () =>
      deepMerge(instance as unknown as Record<string, unknown>, defaultValues),
    [instance, defaultValues]
  );

  const { schema: zodSchema, celDependencyGroups } = useMemo(
    () =>
      buildSectionZodSchema(sectionKey, editSections, {
        formMode: FormMode.Edit,
        originalData: originalDataForCel,
      }),
    [sectionKey, editSections, originalDataForCel]
  );

  const { mutate } = useUpdateDbInstanceWithConflictRetry(instance, {
    onSuccess: () => {
      setSubmitting(false);
      enqueueSnackbar(Messages.onSuccess, { variant: 'success' });
      onSuccess();
      onClose();
    },
    onError: () => {
      setSubmitting(false);
    },
  });

  const handleSubmit = (formData: Record<string, unknown>) => {
    setSubmitting(true);

    const updatedInstance = deepClone(instance) as Instance;
    updatedInstance.spec = mergeSectionEdit({
      spec: updatedInstance.spec as unknown as Record<string, unknown>,
      formData,
      sections,
      sectionKey,
      topology: instance?.spec?.topology?.type,
    }) as Instance['spec'];

    mutate(updatedInstance);
  };

  return (
    <FormDialog
      isOpen
      closeModal={onClose}
      headerMessage={section?.label ?? sectionKey}
      schema={zodSchema}
      celDependencyGroups={celDependencyGroups}
      defaultValues={defaultValues}
      onSubmit={handleSubmit}
      submitMessage={Messages.submitMessage}
      size="XL"
      submitting={submitting}
    >
      <UIGenerator
        sectionKey={sectionKey}
        sections={editSections}
        providerObject={provider}
        formMode={FormMode.Edit}
        namespace={namespace}
        widgetRegistry={widgetRegistry}
      />
    </FormDialog>
  );
};

export default SectionEditModal;
