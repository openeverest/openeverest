import {
  FormControl,
  FormControlLabel,
  FormHelperText,
  Switch,
  Typography,
} from '@mui/material';
import { kebabize } from '@percona/utils';
import { Controller, useFormContext } from 'react-hook-form';
import { SwitchInputProps } from './switch.types';

const SwitchInput = ({
  name,
  control,
  label,
  labelCaption,
  error,
  helperText,
  controllerProps,
  formControlLabelProps,
  formControlProps,
  switchFieldProps = {},
}: SwitchInputProps) => {
  const { control: contextControl } = useFormContext();
  const { onChange, ...restSwitchFieldProps } = switchFieldProps;

  const switchControl = (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field }) => (
        <Switch
          {...field}
          onChange={(e) => {
            onChange?.(e, e.target.checked);
            field.onChange(e);
          }}
          sx={{
            ...(labelCaption && {
              alignSelf: 'flex-start',
              mt: -1,
            }),
          }}
          checked={field.value}
          data-testid={`switch-input-${kebabize(name)}`}
          {...restSwitchFieldProps}
        />
      )}
      {...controllerProps}
    />
  );

  const labelElement = (
    <FormControlLabel
      label={
        <>
          <Typography variant="body1">{label}</Typography>
          {labelCaption && (
            <Typography variant="caption">{labelCaption}</Typography>
          )}
        </>
      }
      data-testid={`switch-input-${kebabize(name)}-label`}
      control={switchControl}
      {...formControlLabelProps}
    />
  );

  const helperTextElement =
    error || helperText ? (
      <FormHelperText error={!!error} sx={{ mx: 0 }}>
        {helperText}
      </FormHelperText>
    ) : null;

  if (formControlProps || helperTextElement) {
    const { sx: formControlSx, ...restFormControlProps } = formControlProps ?? {};

    return (
      <FormControl sx={formControlSx} {...restFormControlProps}>
        {labelElement}
        {helperTextElement}
      </FormControl>
    );
  }

  return labelElement;
};

export default SwitchInput;
