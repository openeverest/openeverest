import { DbType } from '@percona/types';

type Resources = Omit<Record<'cpu' | 'memory' | 'disk', number>, 'disk'> &
  Partial<Pick<Record<'cpu' | 'memory' | 'disk', number>, 'disk'>>;

export type ResourcesTogglesProps = {
  unit?: string;
  unitPlural?: string;
  options: string[];
  sizeOptions: Record<'small' | 'medium' | 'large', Resources>;
  resourceSizePerUnitInputName: string;
  cpuInputName: string;
  diskInputName?: string;
  diskUnitInputName?: string;
  memoryInputName: string;
  numberOfUnitsInputName: string;
  customNrOfUnitsInputName: string;
  disableDiskInput?: boolean;
  allowDiskInputUpdate?: boolean;
  dbType: DbType;
  dbVersion: string;
  disableCustom?: boolean;
  warnForUpscaling?: boolean;
};

export type ResourceInputProps = {
  unit: string;
  unitPlural: string;
  name: string;
  label: string;
  helperText: string;
  endSuffix: string;
  numberOfUnits: number;
  disabled?: boolean;
};
