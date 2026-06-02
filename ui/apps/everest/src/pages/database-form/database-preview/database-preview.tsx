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
  const { sections, sectionsOrder, hasBackupStep } = useDatabaseFormContext();

  // Trigger a re-render when any form value changes so the preview stays in sync
  useWatch();

  const values = getValues();

  const orderedSectionKeys = sectionsOrder || Object.keys(sections);

  const previewSections: {
    stepId: string;
    title: string;
    component: React.ComponentType<DbWizardType>;
  }[] = [
    {
      stepId: BASE_STEP_ID,
      title: 'Basic Information',
      component: PreviewSectionOne,
    },
    ...(showImportStep
      ? [
          {
            stepId: IMPORT_STEP_ID,
            title: 'Import information',
            component: () => <PreviewContentText text="" />,
          },
        ]
      : []),
    ...(hasBackupStep
      ? [
          {
            stepId: BACKUP_STEP_ID,
            title: 'Backups',
            component: PreviewBackupSection,
          },
        ]
      : []),
    ...orderedSectionKeys.map((key) => ({
      stepId: getSectionStepId(key),
      title: sections[key]?.label || key,
      component: (v: DbWizardType) => (
        <DynamicSectionPreview section={sections[key]} formValues={v} />
      ),
    })),
  ];

  return (
    <Stack sx={{ pr: 2, pl: 2, ...sx }} {...stackProps}>
      <Typography variant="overline">{Messages.title}</Typography>
      <Stack>
        {previewSections.map((section, idx) => {
          const Section = section.component;
          return (
            <React.Fragment key={`section-${idx + 1}`}>
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
                sx={{ mt: idx === 0 ? 2 : 0 }}
              >
                <Section {...values} />
              </PreviewSection>
            </React.Fragment>
          );
        })}
      </Stack>
    </Stack>
  );
};
