import { Box, Divider, MenuItem, Tooltip, Typography } from '@mui/material';
import { useFormContext } from 'react-hook-form';
import { SelectInput } from '@percona/ui-lib';
import { AM_PM, HOURS_AM_PM, MINUTES } from '../time-selection.constants';
import { Messages } from '../time-selection.messages';
import { TimeSelectionFields } from '../time-selection.types';
import { addZeroToSingleDigit } from '../time-selection.utils';
import HelpIcon from '@mui/icons-material/Help';

const showLimitationInfo = () => {
  return (
    <Box>
      <MenuItem
        sx={{
          cursor: 'text',
          userSelect: 'text',
          maxHeight: '20px',
          paddingLeft: '5px',
        }}
      >
        <HelpIcon sx={{ size: 'small' }} />

        <Tooltip
          title={Messages.notAvailable}
          arrow
          placement="right"
          sx={{ ml: 1 }}
        >
          <Typography variant="helperText" sx={{
            color: "text.secondary"
          }}>
            {Messages.help}
          </Typography>
        </Tooltip>
      </MenuItem>
      <Divider />
    </Box>
  );
};
export const TimeFields = ({
  selectableHours = HOURS_AM_PM,
  selectableMinutes = MINUTES,
  selectableAmPm = AM_PM,
  shouldRestrictTime,
}: {
  selectableHours?: number[];
  selectableMinutes?: number[];
  selectableAmPm?: string[];
  shouldRestrictTime: boolean;
}) => {
  const { control } = useFormContext();
  return (
    <Box
      sx={(theme) => ({
        display: 'flex',
        alignItems: 'center',
        gap: 2,
        [theme.breakpoints.down('sm')]: {
          flexBasis: '100%',
        },
      })}
    >
      <Typography variant="sectionHeading">{Messages.at}</Typography>
      <SelectInput
        name={TimeSelectionFields.hour}
        control={control}
        selectFieldProps={{
          sx: { minWidth: '80px' },
        }}
      >
        {shouldRestrictTime &&
          HOURS_AM_PM.length !== selectableHours.length &&
          showLimitationInfo()}
        {HOURS_AM_PM.map((value) => (
          <MenuItem
            key={value}
            value={value}
            disabled={!selectableHours.includes(value)}
          >
            {value}
          </MenuItem>
        ))}
      </SelectInput>
      <SelectInput
        name={TimeSelectionFields.minute}
        control={control}
        selectFieldProps={{
          sx: { minWidth: '80px', width: 'fit-content', margin: '5px' },
        }}
      >
        {shouldRestrictTime &&
          MINUTES.length !== selectableMinutes.length &&
          showLimitationInfo()}
        {MINUTES.map((value) => (
          <MenuItem
            key={value}
            value={value}
            disabled={!selectableMinutes.includes(value)}
          >
            {addZeroToSingleDigit(value)}
          </MenuItem>
        ))}
      </SelectInput>
      <SelectInput
        name={TimeSelectionFields.amPm}
        control={control}
        selectFieldProps={{
          sx: { minWidth: '80px' },
        }}
      >
        {shouldRestrictTime &&
          AM_PM.length !== selectableAmPm.length &&
          showLimitationInfo()}
        {AM_PM.map((value) => (
          <MenuItem
            key={value}
            value={value}
            disabled={!selectableAmPm.includes(value)}
          >
            {value}
          </MenuItem>
        ))}
      </SelectInput>
    </Box>
  );
};
