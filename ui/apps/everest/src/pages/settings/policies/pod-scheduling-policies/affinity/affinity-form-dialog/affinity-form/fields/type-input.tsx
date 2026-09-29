import { SelectInput } from '@percona/ui-lib';
import { MenuItem, SxProps, Theme } from '@mui/material';
import { AffinityType, AffinityTypeValue } from 'shared-types/affinity.types';
import { AffinityFormFields } from '../affinity-form.types';

const TypeInput = ({
  sx,
  helperText,
}: {
  sx?: SxProps<Theme>;
  helperText?: string;
}) => (
  <SelectInput
    name={AffinityFormFields.type}
    label={'Type'}
    helperText={helperText}
    selectFieldProps={{ sx: sx ?? { width: '213px' }, label: 'Type' }}
    data-testid="type-select"
  >
    {Object.values(AffinityType).map((value) => (
      <MenuItem key={value} value={value} data-testid={value}>
        {AffinityTypeValue[value]}
      </MenuItem>
    ))}
  </SelectInput>
);

export default TypeInput;
