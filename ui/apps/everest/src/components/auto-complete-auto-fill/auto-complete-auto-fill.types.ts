import { AutoCompleteInputProps } from '@percona/ui-lib';

export type AutoCompleteAutoFillProps<T> = AutoCompleteInputProps<T> & {
  // automatically selects the first available option on mount if the field value is null/undefined
  enableFillFirst?: boolean;
  fillFirstField?: string;
};
