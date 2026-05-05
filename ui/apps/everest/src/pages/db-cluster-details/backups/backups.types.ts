import { Instance } from 'shared-types/api.types';
import { ScheduleWizardMode } from 'shared-types/wizard.types';

export type ScheduleModalContextType = {
  instance: Instance;
  mode: ScheduleWizardMode;
  setMode: React.Dispatch<React.SetStateAction<ScheduleWizardMode>>;
  selectedScheduleName: string;
  setSelectedScheduleName: React.Dispatch<React.SetStateAction<string>>;
  openScheduleModal: boolean;
  setOpenScheduleModal: React.Dispatch<React.SetStateAction<boolean>>;
  openOnDemandModal: boolean;
  setOpenOnDemandModal: React.Dispatch<React.SetStateAction<boolean>>;
};
