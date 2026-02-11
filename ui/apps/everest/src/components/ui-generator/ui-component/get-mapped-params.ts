import { TextFieldProps } from '@mui/material';
import { NumberFieldParams, FieldParamsMap } from '../ui-generator.types';

// Helper to filter out undefined values from an object
const filterDefined = <T extends Record<string, unknown>>(
  obj: T
): Partial<T> => {
  return Object.fromEntries(
    Object.entries(obj).filter(([, value]) => value !== undefined)
  ) as Partial<T>;
};

export const getMappedParams = <K extends keyof FieldParamsMap>(
  fieldType: K,
  fieldParams: FieldParamsMap[K],
  validation?: any
) => {
  switch (fieldType) {
    case 'number':
      return mapNumberFieldParams(fieldParams as NumberFieldParams, validation);
    // Add more cases for other field types as needed
    default:
      return fieldParams;
  }
};

const mapNumberFieldParams = (
  fieldParams: NumberFieldParams,
  validation?: any
) => {
  const {
    step,
    required,
    disabled,
    helperText,
    // badge,
    autoFocus,
    placeholder,
    ...rest
  } = fieldParams;

  const textFieldProps: Partial<TextFieldProps> = filterDefined({
    type: 'number' as const,
    required,
    disabled,
    helperText,
    autoFocus,
    placeholder,
  });

  const inputProps = filterDefined({
    min: validation?.min,
    max: validation?.max,
    step,
  });

  if (Object.keys(inputProps).length > 0) {
    textFieldProps.inputProps = inputProps;
  }

  //TODO custom logic for badge will be added in https://github.com/openeverest/openeverest/issues/1854
  // if (badge) {
  //   textFieldProps.InputProps = {
  //     endAdornment:(<InputAdornment position="end">{badge}</InputAdornment>)
  //   };
  // }

  return {
    ...rest,
    textFieldProps: {
      ...textFieldProps,
      ...(rest || {}),
      inputProps: {
        ...inputProps,
      },
    },
  };
};
