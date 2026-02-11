import { Stack, Typography } from '@mui/material';
import { useEffect } from 'react';
import { useFormContext } from 'react-hook-form';
import { Section } from './ui-generator.types';
import { orderComponents, renderComponent } from './utils/component-renderer';

type UIGeneratorProps = {
  activeStep: number;
  sections: { [key: string]: Section };
  stepLabels: string[];
};

export const UIGenerator = ({
  activeStep,
  sections,
  stepLabels,
}: UIGeneratorProps) => {
  const { getValues, formState } = useFormContext();
  const sectionKey = stepLabels[activeStep];
  const section = sections[sectionKey];
  const components = section?.components;

  useEffect(() => {
    console.log('UIGenerator form state', {
      values: getValues(),
      formState,
      errors: formState.errors,
    });
  }, [getValues, formState]);

  //TODO we can handle different scenarios for section with empty ui components
  //TODO we should add a checking that uiType exists
  if (!components || Object.keys(components).length === 0) {
    return <Typography>No components available for this step</Typography>;
  }

  const orderedComponents = orderComponents(
    components,
    section?.componentsOrder
  );

  // Build base path for field names (no topology key since it's already selected)
  const basePath = sectionKey || '';

  return (
    <Stack spacing={2}>
      {orderedComponents.map(([key, item]) => {
        const fieldName = basePath ? `${basePath}.${key}` : key;
        return renderComponent({
          key,
          item,
          name: fieldName,
        });
      })}
    </Stack>
  );
};
