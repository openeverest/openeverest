import { createContext } from 'react';
import { ScheduleModalContextType } from './backups.types.ts';
import { Instance } from 'shared-types/api.types.ts';
import { WizardMode } from 'shared-types/wizard.types.ts';

export const ScheduleModalContext = createContext<ScheduleModalContextType>({
  instance: {} as Instance,
  openOnDemandModal: false,
  setOpenOnDemandModal: () => {},
  openScheduleModal: false,
  setOpenScheduleModal: () => {},
  mode: WizardMode.New,
  setMode: () => {},
  selectedScheduleName: '',
  setSelectedScheduleName: () => {},
});
