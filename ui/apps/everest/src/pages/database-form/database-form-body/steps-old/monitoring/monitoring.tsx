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

// @ts-nocheck
// TODO remove this file after release of v2
import { FormGroup } from '@mui/material';
import { AutoCompleteInput, SwitchInput } from '@percona/ui-lib';
import { useEffect, useState, useMemo } from 'react';
import {
  MONITORING_INSTANCES_QUERY_KEY,
  useCreateMonitoringInstance,
  useMonitoringInstancesList,
} from 'hooks/api/monitoring/useMonitoringConfigsList.js';
import { CreateEditEndpointModal } from 'pages/settings/monitoring-endpoints/createEditModal/create-edit-modal.tsx';
import { EndpointFormType } from 'pages/settings/monitoring-endpoints/createEditModal/create-edit-modal.types.ts';
import { useFormContext } from 'react-hook-form';
import { useQueryClient } from '@tanstack/react-query';
import { updateDataAfterCreate } from 'utils/generalOptimisticDataUpdate.ts';
import { DbWizardFormFields } from 'consts.ts';
import { useDatabasePageMode } from '../../../hooks/use-database-page-mode.js';
import { StepHeader } from '../step-header/step-header.js';
import { Messages } from './monitoring.messages.js';
import ActionableAlert from 'components/actionable-alert';
import { convertMonitoringInstancesPayloadToTableFormat } from 'pages/settings/monitoring-endpoints/monitoring-endpoints.utils.ts';
import { useRBACPermissions } from 'hooks/rbac';
import { WizardMode } from 'shared-types/wizard.types.ts';

export const Monitoring = () => {
  const [openCreateEditModal, setOpenCreateEditModal] = useState(false);
  const queryClient = useQueryClient();
  const { watch, getValues } = useFormContext();
  const monitoring = watch(DbWizardFormFields.monitoring);
  const selectedNamespace = watch(DbWizardFormFields.k8sNamespace);
  const { canCreate } = useRBACPermissions(
    'monitoring-instances',
    `${selectedNamespace}/*`
  );
  const mode = useDatabasePageMode();
  const { mutate: createMonitoringInstance, isPending: creatingInstance } =
    useCreateMonitoringInstance();
  const { setValue } = useFormContext();

  const monitoringInstancesResult = useMonitoringInstancesList([
    {
      namespace: selectedNamespace,
    },
  ]);

  const monitoringInstancesLoading = monitoringInstancesResult.some(
    (result) => result.queryResult.isLoading
  );

  const monitoringInstances = useMemo(
    () =>
      convertMonitoringInstancesPayloadToTableFormat(monitoringInstancesResult),
    [monitoringInstancesResult]
  );

  const availableMonitoringInstances = useMemo(
    () =>
      (monitoringInstances || []).filter(
        (item) => item.namespace === selectedNamespace
      ),
    [monitoringInstances, selectedNamespace]
  );

  const monitoringInstancesOptions = availableMonitoringInstances.map(
    (item) => item.name
  );
  const getInstanceOptionLabel = (instanceName: string) => {
    const instance = availableMonitoringInstances?.find(
      (inst) => inst.name === instanceName
    );

    return instance ? `${instance.name} (${instance.url})` : '';
  };

  const handleCloseModal = () => {
    setOpenCreateEditModal(false);
  };

  const handleSubmitModal = (
    // @ts-ignore
    _,
    { name, url, namespace, verifyTLS, ...pmmData }: EndpointFormType
  ) => {
    createMonitoringInstance(
      {
        name,
        url,
        type: 'pmm',
        namespace,
        allowedNamespaces: [],
        verifyTLS,
        pmm: { ...pmmData },
      },
      {
        onSuccess: (newInstance) => {
          updateDataAfterCreate(queryClient, [
            MONITORING_INSTANCES_QUERY_KEY,
            newInstance.namespace,
          ])(newInstance);
          handleCloseModal();
        },
      }
    );
  };

  useEffect(() => {
    const selectedInstance = getValues(DbWizardFormFields.monitoringInstance);

    if (mode === WizardMode.New) {
      if (monitoring && availableMonitoringInstances?.length) {
        setValue(
          DbWizardFormFields.monitoringInstance,
          availableMonitoringInstances[0].name,
          { shouldValidate: true }
        );
      }
    }
    if (
      mode === WizardMode.Restore &&
      availableMonitoringInstances?.length &&
      !selectedInstance
    ) {
      setValue(
        DbWizardFormFields.monitoringInstance,
        availableMonitoringInstances[0].name,
        { shouldValidate: true }
      );
    }
  }, [monitoring]);

  return (
    <>
      <StepHeader
        pageTitle={Messages.monitoring}
        pageDescription={Messages.caption}
      />
      {!availableMonitoringInstances?.length && (
        <ActionableAlert
          message={Messages.alertText(selectedNamespace)}
          buttonMessage={Messages.addMonitoringEndpoint}
          data-testid="monitoring-warning"
          onClick={() => setOpenCreateEditModal(true)}
          {...(!canCreate && { action: undefined })}
        />
      )}
      <FormGroup sx={{ mt: 2 }}>
        <SwitchInput
          label={Messages.monitoringEnabled}
          name={DbWizardFormFields.monitoring}
          switchFieldProps={{
            disabled: !availableMonitoringInstances?.length,
          }}
        />
        {monitoring && !!availableMonitoringInstances?.length && (
          <AutoCompleteInput
            name={DbWizardFormFields.monitoringInstance}
            label={Messages.monitoringInstanceLabel}
            loading={monitoringInstancesLoading}
            options={monitoringInstancesOptions}
            autoCompleteProps={{
              disableClearable: true,
              renderOption: (props, option) => (
                <li {...props}>{getInstanceOptionLabel(option)}</li>
              ),
            }}
          />
        )}
        {openCreateEditModal && (
          <CreateEditEndpointModal
            open={openCreateEditModal}
            handleClose={handleCloseModal}
            handleSubmit={handleSubmitModal}
            isLoading={creatingInstance}
          />
        )}
      </FormGroup>
    </>
  );
};
