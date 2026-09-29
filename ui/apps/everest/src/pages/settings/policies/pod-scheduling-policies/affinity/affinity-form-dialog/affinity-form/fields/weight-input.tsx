import { TextInput } from '@percona/ui-lib';
import { SxProps, Theme } from '@mui/material';
import { AffinityFormFields } from '../affinity-form.types';

const WeightInput = ({ sx }: { sx?: SxProps<Theme> }) => (
  <TextInput
    name={AffinityFormFields.weight}
    textFieldProps={{
      helperText: '1 - 100',
      type: 'number',
      sx: sx ?? {
        width: '213px',
        marginTop: '25px',
      },
    }}
    label="Weight"
  />
);

export default WeightInput;
