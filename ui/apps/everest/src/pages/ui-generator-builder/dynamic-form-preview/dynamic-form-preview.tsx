import { FormProvider, useForm } from 'react-hook-form';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { useState, useMemo, useEffect } from 'react';
import { SelectInput, Stepper } from '@percona/ui-lib';
import { TopologyUISchemas } from 'components/ui-generator/ui-generator.types';
import { MenuItem, Stack, Step, StepLabel } from '@mui/material';
import { zodResolver } from '@hookform/resolvers/zod';
import { StepHeader } from 'pages/database-form/database-form-body/steps/step-header/step-header';
import DatabaseFormStepControllers from 'pages/database-form/database-form-body/DatabaseFormStepControllers';
import { getSteps } from 'components/ui-generator/utils/component-renderer';
import { getDefaultValues } from 'components/ui-generator/utils/default-values';
import { buildZodSchema } from 'components/ui-generator/utils/schema-builder';
import { useCelValidation } from 'components/ui-generator/hooks/use-cel-validation';

export type DynamicFormProps = {
  schema: TopologyUISchemas;
};

export const DynamicForm = ({ schema }: DynamicFormProps) => {
  const [activeStep, setActiveStep] = useState(0);
  const topologies = Object.keys(schema);
  const hasMultipleTopologies = topologies.length > 1;
  const defaultTopology = topologies[0] || '';
  const [selectedTopology, setSelectedTopology] =
    useState<string>(defaultTopology);
  const sections = getSteps(selectedTopology, schema);

  // Skip topology selection step if only one topology exists
  const stepLabels = hasMultipleTopologies
    ? ['Choose topology', ...Object.keys(sections)]
    : Object.keys(sections);

  // Generate default values based on unique field IDs
  const defaultValues: Record<string, unknown> = useMemo(() => {
    const values = getDefaultValues(schema, selectedTopology);
    return hasMultipleTopologies
      ? { topology: { type: selectedTopology }, ...values }
      : { topology: { type: defaultTopology }, ...values };
  }, [schema, selectedTopology, hasMultipleTopologies, defaultTopology]);

  // Build Zod validation schema for the selected topology
  const { schema: zodSchema, celDependencyGroups } = buildZodSchema(
    schema,
    selectedTopology
  );

  const methods = useForm<Record<string, unknown>>({
    mode: 'onChange',
    resolver: zodResolver(zodSchema),
    defaultValues,
  });

  // Reset form when topology changes
  useEffect(() => {
    methods.reset(defaultValues);
  }, [selectedTopology, methods, defaultValues]);

  // Enable CEL expression validation with automatic dependency re-validation
  const { trigger, control } = methods;
  useCelValidation(celDependencyGroups, control, trigger);

  return (
    <FormProvider {...methods} key={selectedTopology}>
      <Stepper
        noConnector
        activeStep={activeStep}
        sx={{ marginBottom: 4 }}
        key={`stepper-${selectedTopology}`}
      >
        {stepLabels.map((_, idx) => (
          <Step key={`${selectedTopology}-step-${idx + 1}`}>
            <StepLabel />
          </Step>
        ))}
      </Stepper>
      <Stack spacing={2} sx={{ marginTop: 2 }}>
        <StepHeader
          pageTitle={
            sections[stepLabels[activeStep]]?.label ??
            (stepLabels[activeStep] || '')
          }
          pageDescription={sections[stepLabels[activeStep]]?.description ?? ''}
        />
        {activeStep === 0 && hasMultipleTopologies ? (
          <SelectInput name="topology.type" label="topology type">
            {topologies.map((topKey) => (
              <MenuItem
                value={topKey}
                key={topKey}
                onClick={() => setSelectedTopology(topKey)}
              >
                {topKey}
              </MenuItem>
            ))}
          </SelectInput>
        ) : (
          <UIGenerator
            activeStep={activeStep}
            sections={sections}
            stepLabels={stepLabels}
          />
        )}
      </Stack>
      <DatabaseFormStepControllers
        disableBack={activeStep === 0}
        disableSubmit={
          activeStep !== stepLabels.length - 1 ||
          Object.keys(methods.formState.errors).length > 0
        }
        showSubmit={activeStep === stepLabels.length - 1}
        onPreviousClick={() => setActiveStep((prev) => prev - 1)}
        onNextClick={() => setActiveStep((prev) => prev + 1)}
        onSubmit={() => {}}
        onCancel={() => {}}
      />
    </FormProvider>
  );
};
