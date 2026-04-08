import { FormDialog } from 'components/form-dialog/form-dialog';
import { Messages } from './monitoring.messages';
import {
  EditMonitoringDialogType,
  monitoringEditDialogPropsSchema,
  MonitoringEditDialogProps,
  monitoringEditDialogDefaultValues,
  MonitoringEditDialogFields,
} from './edit-monitoring.types';
import { SubmitHandler } from 'react-hook-form';
import { AutoCompleteInput, SwitchInput } from '@percona/ui-lib';
import { useMonitoringInstancesList } from 'hooks/api/monitoring/useMonitoringInstancesList';
import { useNavigate, useParams } from 'react-router-dom';
import { useMemo } from 'react';
import ActionableAlert from 'components/actionable-alert';
import { convertMonitoringInstancesPayloadToTableFormat } from 'pages/settings/monitoring-endpoints/monitoring-endpoints.utils';

export const MonitoringEditModal = ({
  open,
  handleCloseModal,
  handleSubmitModal,
  monitoringName = '',
  enabled = false,
}: MonitoringEditDialogProps) => {
  const onSubmit: SubmitHandler<EditMonitoringDialogType> = ({
    monitoringNameInput,
    monitoringEnabledInput,
  }) => {
    handleSubmitModal(monitoringNameInput, monitoringEnabledInput);
  };

  const { namespace = '' } = useParams();
  const monitoringInstancesResult = useMonitoringInstancesList([
    {
      namespace,
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
        (item) => item.namespace === namespace
      ),
    [monitoringInstances, namespace]
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

  const navigate = useNavigate();

  const monitoringAvailable = availableMonitoringInstances?.length > 0;

  return (
    <FormDialog
      dataTestId="edit-monitoring"
      size="XL"
      isOpen={open}
      closeModal={handleCloseModal}
      schema={monitoringEditDialogPropsSchema()}
      headerMessage={Messages.editMonitoring}
      onSubmit={onSubmit}
      submitMessage={Messages.save}
      defaultValues={monitoringEditDialogDefaultValues(
        monitoringName || monitoringInstancesOptions[0],
        enabled
      )}
    >
      {({ watch }) => (
        <>
          {!monitoringAvailable && (
            <ActionableAlert
              message={Messages.alertText(namespace)}
              buttonMessage={Messages.addMonitoring}
              data-testid="monitoring-warning"
              onClick={() => navigate('/settings/monitoring-endpoints')}
            />
          )}
          {monitoringAvailable && (
            <>
              <SwitchInput
                label={Messages.enableMonitoring}
                name={MonitoringEditDialogFields.monitoringEnabledInput}
                formControlLabelProps={{
                  sx: {
                    mt: 1,
                  },
                }}
              />
              <AutoCompleteInput
                name={MonitoringEditDialogFields.monitoringNameInput}
                label={Messages.monitoringInputLabel}
                loading={monitoringInstancesLoading}
                options={monitoringInstancesOptions}
                autoCompleteProps={{
                  disableClearable: true,
                  disabled: !watch(
                    MonitoringEditDialogFields.monitoringEnabledInput
                  ),
                  renderOption: (props, option) => (
                    <li {...props}>{getInstanceOptionLabel(option)}</li>
                  ),
                }}
              />
            </>
          )}
        </>
      )}
    </FormDialog>
  );
};
