import { z } from 'zod';

export interface MonitoringEditDialogProps {
  open: boolean;
  handleCloseModal: () => void;
  handleSubmitModal: (monitoringName: string, enabled: boolean) => void;
  monitoringName?: string;
  enabled?: boolean;
}

export enum MonitoringEditDialogFields {
  monitoringNameInput = 'monitoringNameInput',
  monitoringEnabledInput = 'monitoringEnabledInput',
}

export const monitoringEditDialogDefaultValues = (
  monitoringName: string,
  enabled: boolean
) => ({
  [MonitoringEditDialogFields.monitoringNameInput]: monitoringName,
  [MonitoringEditDialogFields.monitoringEnabledInput]: enabled,
});

export const monitoringEditDialogPropsSchema = () =>
  z.object({
    [MonitoringEditDialogFields.monitoringNameInput]: z.string().nonempty(),
    [MonitoringEditDialogFields.monitoringEnabledInput]: z.boolean(),
  });

export type EditMonitoringDialogType = z.infer<
  ReturnType<typeof monitoringEditDialogPropsSchema>
>;
