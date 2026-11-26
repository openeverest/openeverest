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

import type { OutsideFieldError } from './outside-field-errors/outside-field-errors.types';

export const Messages = {
  onSuccess: 'Section updated successfully',
  submitMessage: 'Save',
  outsideErrorsTitle: 'Save is disabled because of fields outside this dialog:',
  outsideError: ({
    fieldId,
    sectionLabel,
    fieldLabel,
    message,
  }: OutsideFieldError) =>
    sectionLabel
      ? `${sectionLabel} › ${fieldLabel ?? fieldId}: ${message}`
      : `${fieldId}: ${message}`,
};
