// everest
// Copyright (C) 2023 Percona LLC
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

import React from 'react';
import { Box } from '@mui/material';
import { useDatabasePageMode } from '../hooks/use-database-page-mode';
import { useDatabasePageDefaultValues } from '../hooks/use-database-form-default-values';
import { DatabaseFormBodyProps } from './types';
import { useFormContext } from 'react-hook-form';
import { WizardMode } from 'shared-types/wizard.types';
import { useDatabaseFormContext } from '../database-form-context';
import { StepHeader } from './steps-old/step-header/step-header';
import DatabaseFormStepControllers from './database-form-step-controllers';

const DatabaseFormBody = ({
  steps,
  activeStep,
  isSubmitting,
  disableNext,
  onCancel,
  onSubmit,
  handleNextStep,
  handlePreviousStep,
}: DatabaseFormBodyProps) => {
  const isFirstStep = activeStep === 0;
  const isLastStep = activeStep === steps.length - 1;
  const mode = useDatabasePageMode();
  const { uiSchema, defaultTopology } = useDatabaseFormContext();
  const {
    formState: { isValid },
  } = useFormContext();

  const { dbClusterRequestStatus, isFetching: loadingDefaultsForEdition } =
    useDatabasePageDefaultValues(mode, uiSchema, defaultTopology);

  const currentStep = steps[activeStep];

  return (
    <form style={{ flexGrow: 1 }} onSubmit={onSubmit}>
      {activeStep > 0 && currentStep?.description && (
        <StepHeader
          pageTitle={currentStep.label}
          pageDescription={currentStep.description}
        />
      )}
      {activeStep > 0 &&
        !currentStep?.description &&
        currentStep?.sectionKey && (
          <StepHeader pageTitle={currentStep.label} pageDescription="" />
        )}
      <Box>
        {(mode === WizardMode.New ||
          (mode === WizardMode.Restore &&
            dbClusterRequestStatus === 'success')) &&
          currentStep &&
          React.createElement(currentStep.component, {
            loadingDefaultsForEdition,
          })}
      </Box>
      <DatabaseFormStepControllers
        disableBack={isFirstStep}
        disableSubmit={isSubmitting || !isValid}
        disableCancel={isSubmitting}
        disableNext={disableNext}
        showSubmit={isLastStep || isFirstStep}
        showConfigMore={isFirstStep}
        onPreviousClick={handlePreviousStep}
        onNextClick={handleNextStep}
        onCancel={onCancel}
        onSubmit={onSubmit}
      />
    </form>
  );
};

export default DatabaseFormBody;
