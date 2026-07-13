import { Box, FormControlLabel, Tooltip } from '@mui/material';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { CheckboxInput } from '@percona/ui-lib';

type S3CheckboxProps = {
  name: string;
  text: string;
  tooltip: string;
};

export const S3Checkbox = ({ name, text, tooltip }: S3CheckboxProps) => {
  return (
    <FormControlLabel
      label={
        <Box
          sx={{
            display: "flex",
            mt: 0
          }}>
          {text}
          <Tooltip title={tooltip} arrow placement="right" sx={{ ml: 1 }}>
            <InfoOutlinedIcon />
          </Tooltip>
        </Box>
      }
      control={<CheckboxInput name={name} />}
    />
  );
};
