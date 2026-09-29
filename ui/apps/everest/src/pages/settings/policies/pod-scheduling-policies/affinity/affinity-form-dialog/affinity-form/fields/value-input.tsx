import { TextInput } from '@percona/ui-lib';
import { SxProps, Theme } from '@mui/material';
import { AffinityFormFields } from '../affinity-form.types';

type Props = {
  disabled: boolean;
  namePrefix?: string;
  sx?: SxProps<Theme>;
};

const ValueInput = ({ disabled, namePrefix = '', sx }: Props) => (
  <TextInput
    name={`${namePrefix}${AffinityFormFields.values}`}
    label={'Values'}
    textFieldProps={{
      sx: sx ?? {
        marginTop: '25px',
        width: '645px',
      },
      inputProps: {
        disabled,
      },
      helperText: 'Insert comma seperated values',
    }}
  />
);

export default ValueInput;
