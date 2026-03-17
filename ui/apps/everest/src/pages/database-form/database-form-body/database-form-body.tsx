// everest
// Copyright (C) 2023 Percona LLC
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
import { useSteps } from './steps';
import { useDatabaseFormContext } from '../database-form-context';
import { StepHeader } from './steps-old/step-header/step-header';
import DatabaseFormStepControllers from './database-form-step-controllers';

const DatabaseFormBody = ({
  activeStep,
  isSubmitting,
  disableNext,
  onCancel,
  onSubmit,
  handleNextStep,
  handlePreviousStep,
}: DatabaseFormBodyProps) => {
  const mode = useDatabasePageMode();
  const { uiSchema, defaultTopology, sections, sectionsOrder, providerObject } =
    useDatabaseFormContext();
  const {
    formState: { isValid },
  } = useFormContext();
  const steps = useSteps(sections, sectionsOrder, providerObject);

  const { dbClusterRequestStatus, isFetching: loadingDefaultsForEdition } =
    useDatabasePageDefaultValues(mode, uiSchema, defaultTopology);

  const isFirstStep = activeStep === 0;

  const sectionKeys = sectionsOrder || Object.keys(sections);
  const stepLabel = steps[activeStep].label;
  const sectionKey = sectionKeys.find(
    (key) => sections[key]?.label === stepLabel
  );
  const sectionInfo = sectionKey ? sections[sectionKey] : null;

  //TODO
  // const isLastStep = activeStep === steps.length - 1;

  return (
    <form style={{ flexGrow: 1 }} onSubmit={onSubmit}>
      {activeStep > 0 && sectionInfo && (
        <StepHeader
          pageTitle={sectionInfo.label || stepLabel}
          pageDescription={sectionInfo.description || ''}
        />
      )}
      <Box>
        {(mode === WizardMode.New ||
          (mode === WizardMode.Restore &&
            dbClusterRequestStatus === 'success')) &&
          React.createElement(steps[activeStep].component, {
            loadingDefaultsForEdition,
          })}
      </Box>
      <DatabaseFormStepControllers
        disableBack={isFirstStep}
        disableSubmit={isSubmitting || !isValid}
        disableCancel={isSubmitting}
        disableNext={disableNext}
        showSubmit={activeStep === steps.length - 1 || activeStep === 0}
        showConfigMore={activeStep === 0}
        onPreviousClick={handlePreviousStep}
        onNextClick={handleNextStep}
        onCancel={onCancel}
        onSubmit={onSubmit}
      />
    </form>
  );
};

export default DatabaseFormBody;
