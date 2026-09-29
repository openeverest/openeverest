import { TextInput } from '@percona/ui-lib';
import { SxProps, Theme, Tooltip } from '@mui/material';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { AffinityFormFields } from '../affinity-form.types';
import { AffinityType } from 'shared-types/affinity.types';
import { Messages } from '../../affinity-form-dialog.messages';

const KeyInput = ({
  affinityType,
  namePrefix = '',
  sx,
  helperInAdornment = false,
  placeholder,
}: {
  affinityType: AffinityType;
  namePrefix?: string;
  sx?: SxProps<Theme>;
  helperInAdornment?: boolean;
  placeholder?: string;
}) => {
  const helperText = Messages.affinityTypeHelperText(affinityType);

  return (
    <TextInput
      name={`${namePrefix}${AffinityFormFields.key}`}
      label="Key"
      // Deps allows RHF to trigger cross-validation on dependent fields
      controllerProps={{
        rules: {
          deps: [
            `${namePrefix}${AffinityFormFields.operator}`,
            `${namePrefix}${AffinityFormFields.values}`,
          ],
        },
      }}
      textFieldProps={{
        placeholder,
        sx: sx ?? {
          flex: '0 0 35%',
        },
        ...(helperInAdornment
          ? {
              InputProps: {
                endAdornment: (
                  <Tooltip title={helperText} placement="top" arrow>
                    <InfoOutlinedIcon
                      sx={{ width: 18, color: 'action.active' }}
                    />
                  </Tooltip>
                ),
              },
            }
          : { helperText }),
      }}
    />
  );
};
export default KeyInput;
