import {
  AffinityOperator,
  AffinityOperatorValue,
} from 'shared-types/affinity.types';
import { AffinityFormFields } from '../affinity-form.types';
import { SelectInput } from '@percona/ui-lib';
import { MenuItem, SxProps, Theme } from '@mui/material';

type Props = {
  disabled: boolean;
  namePrefix?: string;
  sx?: SxProps<Theme>;
};

const OperatorInput = ({ disabled, namePrefix = '', sx }: Props) => (
  <SelectInput
    name={`${namePrefix}${AffinityFormFields.operator}`}
    label="Operator"
    selectFieldProps={{
      sx: sx ?? { width: '213px' },
      label: 'Operator',
      disabled,
    }}
    data-testid="operator-select"
  >
    {Object.values(AffinityOperator).map((value) => (
      <MenuItem key={value} value={value} data-testid={value}>
        {AffinityOperatorValue[value]}
      </MenuItem>
    ))}
  </SelectInput>
);

export default OperatorInput;
