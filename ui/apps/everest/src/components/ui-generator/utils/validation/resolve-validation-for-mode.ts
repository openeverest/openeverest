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

import type {
  CommonValidation,
  FormMode,
  ModeAwareValidation,
} from '../../ui-generator.types';

export const resolveValidationForMode = <T extends CommonValidation>(
  validation: ModeAwareValidation<T> | undefined,
  formMode?: FormMode
): T | undefined => {
  if (!validation) return undefined;

  // Separate the mode-specific branches from the base validation props
  const { modes, ...baseProps } = validation;
  const base =
    Object.keys(baseProps).length > 0 ? (baseProps as unknown as T) : undefined;
  const modeBranch = formMode ? modes?.[formMode] : undefined;

  // No mode branch: return base as-is
  if (!modeBranch) return base;

  const { inheritShared = true, celExpressions, ...modeScalars } = modeBranch;

  // If not inheriting base, mode branch is the entire result
  if (!inheritShared) {
    return { celExpressions, ...modeScalars } as unknown as T;
  }

  // Start from base, overlay scalars from mode branch
  const merged = { ...base, ...modeScalars } as Record<string, unknown>;

  // Append mode celExpressions to base celExpressions
  const baseCel = (base as CommonValidation)?.celExpressions ?? [];
  const modeCel = celExpressions ?? [];
  const combinedCel = [...baseCel, ...modeCel];

  if (combinedCel.length > 0) {
    merged.celExpressions = combinedCel;
  } else {
    delete merged.celExpressions;
  }

  return merged as unknown as T;
};
