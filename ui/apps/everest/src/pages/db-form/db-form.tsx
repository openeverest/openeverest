import { useState } from 'react';
//TODO customise schema by operator type
import { topologyUiSchemas } from '../../components/ui-generator/ui-generator.mock';
import { MenuItem, Stack, Step, StepLabel } from '@mui/material';
import { SelectInput, Stepper } from '@percona/ui-lib';
import DatabaseFormStepControllers from 'pages/database-form/database-form-body/DatabaseFormStepControllers';
import { FormProvider, useForm } from 'react-hook-form';
import { StepHeader } from 'pages/database-form/database-form-body/steps/step-header/step-header';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { getSteps } from 'components/ui-generator/utils/component-renderer';

export const DatabasePageGenerated = () => {
  const [activeStep, setActiveStep] = useState(0);
  const [selectedTopology, setSelectedTopology] = useState<string>('');
  const sections = getSteps(selectedTopology, topologyUiSchemas);
  const stepLabels = ['Choose topology', ...Object.keys(sections)];

  // const selectedTopology = 'replica';
  const topologies = Object.keys(topologyUiSchemas);

  const methods = useForm({
    mode: 'onChange',
    // resolver: async (data, context, options) => {
    // //   const customResolver = zodResolver(schema);
    //   const result = await customResolver(data, context, options);
    //   return result;
    // },
    // defaultValues,
  });

  //TODO: set defaults for topology after selection (check how it works in builder)
  //TODO: check StepHeader empty fields

  return (
    <FormProvider {...methods}>
      <Stepper noConnector activeStep={activeStep} sx={{ marginBottom: 4 }}>
        {stepLabels.map((_, idx) => (
          <Step key={`step-${idx + 1}`}>
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
        {activeStep === 0 ? (
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
