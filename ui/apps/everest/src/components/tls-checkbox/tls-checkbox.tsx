import { Box, FormControlLabel, Tooltip } from '@mui/material';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { CheckboxInput } from '@percona/ui-lib';
import { TlsCheckboxProps } from './tls-checkbox.types';
import { Messages } from './tls-checkbox.messages';

const TlsCheckbox = ({ formControlLabelProps }: TlsCheckboxProps) => (
  <FormControlLabel
    {...formControlLabelProps}
    label={
      <Box
        sx={{
          display: "flex",
          mt: 0
        }}>
        {Messages.verifyTLS}
        <Tooltip
          title={Messages.tooltip}
          arrow
          placement="right"
          sx={{ ml: 1 }}
        >
          <InfoOutlinedIcon />
        </Tooltip>
      </Box>
    }
    control={<CheckboxInput name="verifyTLS" />}
  />
);

export default TlsCheckbox;
