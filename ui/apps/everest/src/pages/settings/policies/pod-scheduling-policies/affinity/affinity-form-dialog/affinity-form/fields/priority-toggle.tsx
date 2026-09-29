import { ToggleButtonGroupInput, ToggleCard } from '@percona/ui-lib';
import { SxProps, Theme } from '@mui/material';
import {
  AffinityPriority,
  AffinityPriorityValue,
} from 'shared-types/affinity.types';
import { AffinityFormFields } from '../affinity-form.types';

const PriorityToggle = ({ sx }: { sx?: SxProps<Theme> }) => (
  <ToggleButtonGroupInput // TODO needs extra styling to look like FIGMA
    name={AffinityFormFields.priority}
    toggleButtonGroupProps={{
      size: 'small',
      sx: sx ?? {
        height: '30px',
        width: '160px',
        marginTop: '20px',
        alignSelf: 'center',
      },
    }}
  >
    {Object.values(AffinityPriority).map((value) => (
      <ToggleCard
        sx={{ borderRadius: '15px' }}
        value={value}
        data-testid={`toggle-button-${value}`}
        key={value}
      >
        {AffinityPriorityValue[value]}
      </ToggleCard>
    ))}
  </ToggleButtonGroupInput>
);
export default PriorityToggle;
