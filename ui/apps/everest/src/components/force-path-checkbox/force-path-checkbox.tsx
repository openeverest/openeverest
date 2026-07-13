import { Box, FormControlLabel, Tooltip } from '@mui/material';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { CheckboxInput } from '@percona/ui-lib';
import { ForcePathCheckboxProps } from './force-path-checkbox.types';
import { Messages } from './force-path-checkbox.messages';

const ForcePathCheckbox = ({
  formControlLabelProps,
}: ForcePathCheckboxProps) => (
  <FormControlLabel
    {...formControlLabelProps}
    label={
      <Box
        sx={{
          display: "flex",
          mt: 0
        }}>
        {Messages.forcePathStyle}
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
    control={<CheckboxInput name="forcePathStyle" />}
  />
);

export default ForcePathCheckbox;
