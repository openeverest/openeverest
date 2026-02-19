import { Stack, Typography } from '@mui/material';
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
  const sectionKey = stepLabels[activeStep];
  const section = sections[sectionKey];
  const components = section?.components;

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
          item,
          name: fieldName,
        });
      })}
    </Stack>
  );
};
