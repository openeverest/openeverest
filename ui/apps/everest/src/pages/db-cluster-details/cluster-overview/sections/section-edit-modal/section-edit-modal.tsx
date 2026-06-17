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
import {
  FormMode,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { buildSectionZodSchema } from 'components/ui-generator/utils/schema-builder';
import { extractInstanceValues } from 'components/ui-generator/utils/default-values/extract-instance-values';
import { applyModeOverrides } from 'components/ui-generator/utils/preprocess/apply-mode-overrides';
import { postprocessSchemaData } from 'components/ui-generator/utils/postprocess/postprocess-schema';
import {
  deepClone,
  deepMerge,
} from 'components/ui-generator/utils/object-path/object-path';
import {
  extractBadgeMappingsFromSections,
  stripBadgesFromData,
} from 'components/ui-generator/utils/badge-to-api/badge-to-api';
import { useUpdateDbInstanceWithConflictRetry } from 'hooks/api/db-instances/useUpdateDbInstance';
import { useKubernetesClusterInfo } from 'hooks/api/kubernetesClusters/useKubernetesClusterInfo';
import type { Instance } from 'shared-types/api.types';
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

  // Strip badge suffixes (e.g. "25Gi" → 25) from the original instance so that
  // CEL numeric comparisons like `size >= original.size` work correctly.
  const originalDataForCel = useMemo(() => {
    const badgeMappings = extractBadgeMappingsFromSections(editSections);
    return stripBadgesFromData(
      instance as unknown as Record<string, unknown>,
      badgeMappings
    );
  }, [editSections, instance]);

  const { schema: zodSchema, celDependencyGroups } = useMemo(
    () =>
      buildSectionZodSchema(sectionKey, editSections, {
        formMode: FormMode.Edit,
        originalData: originalDataForCel,
      }),
    [sectionKey, editSections, originalDataForCel]
  );

  const defaultValues = useMemo(
    () =>
      extractInstanceValues(
        editSections,
        instance as unknown as Record<string, unknown>,
        FormMode.Edit
      ),
    [editSections, instance]
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

    const topologyType = instance?.spec?.topology?.type;
    const uiSchema = provider?.spec?.uiSchema as TopologyUISchemas | undefined;

    const processed = postprocessSchemaData(formData, {
      schema: uiSchema,
      selectedTopology: topologyType,
    });

    const updatedInstance = deepClone(instance) as Instance;
    const specUpdates = (processed as Record<string, unknown>).spec;

    if (specUpdates && typeof specUpdates === 'object') {
      updatedInstance.spec = deepMerge(
        updatedInstance.spec as unknown as Record<string, unknown>,
        specUpdates as Record<string, unknown>
      ) as Instance['spec'];
    }

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
      />
    </FormDialog>
  );
};

export default SectionEditModal;
