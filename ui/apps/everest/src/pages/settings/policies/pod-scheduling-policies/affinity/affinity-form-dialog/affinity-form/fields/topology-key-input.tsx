import { TextInput } from '@percona/ui-lib';
import { SxProps, Theme } from '@mui/material';
import { AffinityFormFields } from '../affinity-form.types';

const TopologyKeyInput = ({ sx }: { sx?: SxProps<Theme> }) => (
  <TextInput
    name={AffinityFormFields.topologyKey}
    label="Topology Key"
    textFieldProps={{
      sx: sx ?? {
        flex: '0 0 35%',
      },
      helperText: 'A domain key that determines relative pod placement',
    }}
  />
);

export default TopologyKeyInput;
