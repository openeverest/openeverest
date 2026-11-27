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

import { Alert, Typography } from '@mui/material';
import { Messages } from '../section-edit-modal.messages';
import type { OutsideFieldErrorsProps } from './outside-field-errors.types';
import { useOutsideFieldErrors } from './use-outside-field-errors';

export const OutsideFieldErrors = ({
  sectionKey,
  sections,
}: OutsideFieldErrorsProps) => {
  const outsideErrors = useOutsideFieldErrors(sectionKey, sections);

  if (outsideErrors.length === 0) {
    return null;
  }

  return (
    <Alert
      severity="error"
      sx={{ mb: 1 }}
      data-testid="section-outside-field-errors"
    >
      <Typography variant="body2">{Messages.outsideErrorsTitle}</Typography>
      {outsideErrors.map((error) => (
        <Typography key={error.fieldId} variant="body2">
          {Messages.outsideError(error)}
        </Typography>
      ))}
    </Alert>
  );
};
