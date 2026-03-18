// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { FormProvider, useForm } from 'react-hook-form';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { useState, useEffect } from 'react';
import { SelectInput, Stepper } from '@percona/ui-lib';
import { TopologyUISchemas } from 'components/ui-generator/ui-generator.types';
import { MenuItem, Stack, Step, StepLabel } from '@mui/material';
import { zodResolver } from '@hookform/resolvers/zod';
import { StepHeader } from 'pages/database-form/database-form-body/steps-old/step-header/step-header';
import DatabaseFormStepControllers from 'pages/database-form/database-form-body/database-form-step-controllers/database-form-step-controllers';
import { useCelValidation } from 'components/ui-generator/hooks/use-cel-validation';
import { useUiGenerator } from 'components/ui-generator/hooks/ui-generator';
import { useTopology } from '../../../components/ui-generator/hooks/use-topology';
import { useDefaultValues } from 'components/ui-generator/hooks/use-default-values';

export type DynamicFormProps = {
  schema: TopologyUISchemas;
};

export const DynamicForm = ({ schema }: DynamicFormProps) => {
  const [activeStep, setActiveStep] = useState(0);
  const {
    selectedTopology,
    setSelectedTopology,
    topologies,
    hasMultipleTopologies,
  } = useTopology(schema);
  const defaultValues = useDefaultValues(schema, selectedTopology);
  //TODO probably it's better to move default values under useUiGenerator
  const {
    sections,
    zodSchema: { schema: zodSchema, celDependencyGroups },
  } = useUiGenerator(schema, selectedTopology);

  const stepLabels = hasMultipleTopologies
    ? ['Choose topology', ...Object.keys(sections)]
    : Object.keys(sections);

  // Build Zod validation schema for the selected topology

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

  // Trigger validation when step changes to enable/disable submit button
  useEffect(() => {
    void methods.trigger();
  }, [activeStep, methods]);

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
      <Stack key="form-content" spacing={2} sx={{ marginTop: 2 }}>
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
            sectionKey={stepLabels[activeStep]}
            sections={sections}
          />
        )}
      </Stack>
      <DatabaseFormStepControllers
        key="form-controllers"
        disableBack={activeStep === 0}
        disableSubmit={
          activeStep !== stepLabels.length - 1 || !methods.formState.isValid
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
