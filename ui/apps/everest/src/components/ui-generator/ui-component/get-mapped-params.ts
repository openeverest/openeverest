import { TextFieldProps } from '@mui/material';
import {
  NumberFieldParams,
  FieldParamsMap,
  ValidationMap,
  FieldType,
} from '../ui-generator.types';

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
  validation?: ValidationMap[K]
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
  validation?: ValidationMap[FieldType.Number]
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

  const getOffset = (): number => {
    if (validation?.int) {
      return 1;
    }
    if (step !== undefined) {
      return step;
    }
    return 0.000001;
  };

  const offset = getOffset();

  // Priority: explicit min/max > converted gt/lt
  const minValue =
    validation?.min !== undefined
      ? validation.min
      : validation?.gt !== undefined
        ? validation.gt + offset
        : undefined;

  const maxValue =
    validation?.max !== undefined
      ? validation.max
      : validation?.lt !== undefined
        ? validation.lt - offset
        : undefined;

  const inputProps = filterDefined({
    min: minValue,
    max: maxValue,
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
      inputProps: {
        ...inputProps,
      },
    },
  };
};
