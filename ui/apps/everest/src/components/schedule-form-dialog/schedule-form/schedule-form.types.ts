import { TimeSelectionFields } from '../../time-selection/time-selection.types';
import { Schedule } from 'shared-types/dbCluster.types';

enum ScheduleForm {
  scheduleName = 'scheduleName',
  storageLocation = 'storageLocation',
  retentionCopies = 'retentionCopies',
}

export type ScheduleFormProps = {
  allowScheduleSelection?: boolean;
  disableStorageSelection?: boolean;
  autoFillLocation?: boolean;
  disableNameInput?: boolean;
  schedules: Schedule[];
  showTypeRadio: boolean;
  disableNameEdit?: boolean;
  maxSchedulesPerStorage?: number;
};

export const ScheduleFormFields = { ...ScheduleForm, ...TimeSelectionFields };
export type ScheduleFormFields = ScheduleForm | TimeSelectionFields;
