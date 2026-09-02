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

import React, { useMemo } from 'react';
import { Stack, Typography } from '@mui/material';
import { useFormContext, useWatch } from 'react-hook-form';
import { useLocation } from 'react-router-dom';
import { DatabasePreviewProps } from './database-preview.types.ts';
import { Messages } from './database.preview.messages.ts';
import { PreviewSection, PreviewContentText } from './preview-section.tsx';
import { DbWizardType } from '../database-form-schema.ts';
import { useDatabaseFormContext } from '../database-form-context.tsx';
import DynamicSectionPreview from './dynamic-section-preview/dynamic-section-preview.tsx';
import { PreviewSectionOne } from './sections/base-step.tsx';
import { PreviewBackupSection } from './sections/backup-section.tsx';
import {
  BASE_STEP_ID,
  BACKUP_STEP_ID,
  IMPORT_STEP_ID,
} from '../database-form-body/steps/constants.ts';
import { getSectionStepId } from 'components/ui-generator/utils/section-step-id.ts';
import {
  mergeWizardSteps,
  StepDefinition,
} from 'components/ui-generator/form-engine';

export const DatabasePreview = ({
  activeStepId,
  onSectionEdit = () => {},
  disabled,
  stepsWithErrors,
  sx,
  ...stackProps
}: DatabasePreviewProps) => {
  const { getValues } = useFormContext<DbWizardType>();
  const location = useLocation();
  const showImportStep = location.state?.showImport;
  const { sections, sectionsOrder, hasBackupStep, steps } =
    useDatabaseFormContext();

  // Trigger a re-render when any form value changes so the preview stays in sync
  useWatch();

  const values = getValues();

  const fallbackSteps = useMemo((): StepDefinition[] => {
    if (steps) {
      return steps;
    }

    const fallbackStaticSteps: StepDefinition[] = [
      {
        id: BASE_STEP_ID,
        label: 'Basic Information',
        component: () => null,
        fields: [],
      },
    ];

    if (showImportStep) {
      fallbackStaticSteps.push({
        id: IMPORT_STEP_ID,
        label: 'Import information',
        component: () => null,
        fields: [],
      });
    }

    if (hasBackupStep) {
      fallbackStaticSteps.push({
        id: BACKUP_STEP_ID,
        label: 'Backups',
        component: () => null,
        fields: [],
      });
    }

    const genMap = new Map<string, StepDefinition>();
    for (const key of Object.keys(sections)) {
      genMap.set(key, {
        id: getSectionStepId(key),
        label: sections[key]?.label || key,
        sectionKey: key,
        component: () => null,
        fields: [],
      });
    }

    return mergeWizardSteps(fallbackStaticSteps, genMap, sectionsOrder);
  }, [steps, showImportStep, hasBackupStep, sections, sectionsOrder]);

  const effectiveSteps = steps || fallbackSteps;

  const previewSections: {
    stepId: string;
    title: string;
    content: React.ReactNode;
  }[] = effectiveSteps.map((step) => {
    if (step.id === BASE_STEP_ID) {
      return {
        stepId: BASE_STEP_ID,
        title: 'Basic Information',
        content: <PreviewSectionOne {...values} />,
      };
    }
    if (step.id === IMPORT_STEP_ID) {
      return {
        stepId: IMPORT_STEP_ID,
        title: 'Import information',
        content: <PreviewContentText text="" />,
      };
    }
    if (step.id === BACKUP_STEP_ID) {
      return {
        stepId: BACKUP_STEP_ID,
        title: 'Backups',
        content: <PreviewBackupSection {...values} />,
      };
    }
    const sectionKey = step.sectionKey;
    const section = sectionKey ? sections[sectionKey] : undefined;
    return {
      stepId: step.id,
      title: section?.label || step.label || step.id,
      content: section ? (
        <DynamicSectionPreview section={section} formValues={values} />
      ) : null,
    };
  });

  return (
    <Stack
      sx={[{ pr: 2, pl: 2 }, ...(Array.isArray(sx) ? sx : [sx])]}
      {...stackProps}
    >
      <Typography variant="overline">{Messages.title}</Typography>
      <Stack>
        {previewSections.map((section, idx) => {
          return (
            <React.Fragment key={section.stepId}>
              <PreviewSection
                order={idx + 1}
                title={section.title}
                hasBeenReached
                hasError={
                  stepsWithErrors.includes(section.stepId) &&
                  activeStepId !== section.stepId
                }
                active={activeStepId === section.stepId}
                disabled={disabled}
                onEditClick={() => onSectionEdit(section.stepId)}
                sx={[
                  idx === 0
                    ? {
                        mt: 2,
                      }
                    : {
                        mt: 0,
                      },
                ]}
              >
                {section.content}
              </PreviewSection>
            </React.Fragment>
          );
        })}
      </Stack>
    </Stack>
  );
};
