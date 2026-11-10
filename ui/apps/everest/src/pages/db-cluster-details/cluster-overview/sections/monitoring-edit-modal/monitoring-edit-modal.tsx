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
import { FormDialog } from 'components/form-dialog';
import { getByPath } from 'components/ui-generator/utils/object-path';
import { useUpdateDbInstanceWithConflictRetry } from 'hooks/api/db-instances/useUpdateDbInstance';
import {
  buildMonitoringParametersSchema,
  finalizeMonitoringEntries,
  MONITORING_DESTINATIONS_FIELD,
  MonitoringDestinationsPicker,
  parseMonitoringDestinations,
  useMonitoringInstanceForms,
} from 'pages/database-form/database-form-body/steps/monitoring-step';
import { DisruptionNotice } from './disruption-notice/disruption-notice';
import { Messages } from './monitoring-edit-modal.messages';
import { MonitoringEditModalProps } from './monitoring-edit-modal.types';
import { withMonitoringDestinations } from './monitoring-edit-modal.utils';

// Adds and removes the Instance's monitoring destinations with the wizard's
// picker; the Instance is updated with a conflict retry.
export const MonitoringEditModal = ({
  instance,
  namespace,
  onClose,
}: MonitoringEditModalProps) => {
  const provider = instance.spec?.providerRef.name ?? '';
  const [submitting, setSubmitting] = useState(false);
  const instanceForms = useMonitoringInstanceForms(namespace, provider);
  const schema = useMemo(
    () => buildMonitoringParametersSchema(instanceForms),
    [instanceForms]
  );
  const defaultValues = useMemo(
    () => ({
      monitoring: {
        destinations: instance.spec?.monitoring?.destinations ?? [],
      },
    }),
    [instance]
  );

  const { mutate } = useUpdateDbInstanceWithConflictRetry(instance, {
    onSuccess: () => {
      setSubmitting(false);
      enqueueSnackbar(Messages.saved, { variant: 'success' });
      onClose();
    },
    onError: () => setSubmitting(false),
  });

  const handleSubmit = (values: Record<string, unknown>) => {
    setSubmitting(true);
    const entries = parseMonitoringDestinations(
      getByPath(values, MONITORING_DESTINATIONS_FIELD)
    );
    mutate(
      withMonitoringDestinations(
        instance,
        finalizeMonitoringEntries(entries, instanceForms)
      )
    );
  };

  return (
    <FormDialog<Record<string, unknown>>
      isOpen
      closeModal={onClose}
      headerMessage={Messages.title}
      schema={schema}
      defaultValues={defaultValues}
      onSubmit={handleSubmit}
      submitting={submitting}
      submitMessage={Messages.save}
      dataTestId="monitoring-edit"
      size="XXXL"
    >
      <MonitoringDestinationsPicker
        namespace={namespace}
        provider={provider}
        pruneUnavailable={false}
      />
      <DisruptionNotice instance={instance} namespace={namespace} />
    </FormDialog>
  );
};
