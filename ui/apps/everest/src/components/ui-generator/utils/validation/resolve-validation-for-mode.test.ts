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

import { describe, it, expect } from 'vitest';
import { resolveValidationForMode } from './resolve-validation-for-mode';
import { FormMode } from '../../ui-generator.types';
import type {
  ModeAwareValidation,
  NumberValidation,
} from '../../ui-generator.types';

describe('resolveValidationForMode', () => {
  it('returns undefined when validation is undefined', () => {
    expect(resolveValidationForMode(undefined, FormMode.New)).toBeUndefined();
  });

  it('returns base props when mode branch does not exist', () => {
    const v: ModeAwareValidation<NumberValidation> = {
      required: true,
      min: 1,
    };
    expect(resolveValidationForMode(v, FormMode.Edit)).toEqual({
      required: true,
      min: 1,
    });
  });

  it('returns base props when formMode is undefined', () => {
    const v: ModeAwareValidation<NumberValidation> = {
      required: true,
      min: 1,
      modes: {
        [FormMode.Edit]: { min: 5 },
      },
    };
    expect(resolveValidationForMode(v, undefined)).toEqual({
      required: true,
      min: 1,
    });
  });

  it('merges scalar mode overrides on top of base', () => {
    const v: ModeAwareValidation<NumberValidation> = {
      required: true,
      min: 1,
      int: true,
      modes: {
        [FormMode.Edit]: { min: 5 },
      },
    };
    expect(resolveValidationForMode(v, FormMode.Edit)).toEqual({
      required: true,
      min: 5,
      int: true,
    });
  });

  it('appends mode celExpressions to base celExpressions', () => {
    const v: ModeAwareValidation<NumberValidation> = {
      min: 1,
      celExpressions: [{ celExpr: 'x % 2 == 1', message: 'must be odd' }],
      modes: {
        [FormMode.Edit]: {
          celExpressions: [
            {
              celExpr: 'x >= original.x',
              message: 'cannot decrease',
            },
          ],
        },
      },
    };

    const resolved = resolveValidationForMode(v, FormMode.Edit);
    expect(resolved?.celExpressions).toHaveLength(2);
    expect(resolved?.celExpressions?.[0].celExpr).toBe('x % 2 == 1');
    expect(resolved?.celExpressions?.[1].celExpr).toBe('x >= original.x');
  });

  it('does not include edit celExpressions in new mode', () => {
    const v: ModeAwareValidation<NumberValidation> = {
      min: 1,
      celExpressions: [{ celExpr: 'x % 2 == 1', message: 'must be odd' }],
      modes: {
        [FormMode.Edit]: {
          celExpressions: [
            { celExpr: 'x >= original.x', message: 'cannot decrease' },
          ],
        },
      },
    };

    const resolved = resolveValidationForMode(v, FormMode.New);
    expect(resolved?.celExpressions).toHaveLength(1);
    expect(resolved?.celExpressions?.[0].celExpr).toBe('x % 2 == 1');
  });

  it('uses only mode branch when inheritShared is false', () => {
    const v: ModeAwareValidation<NumberValidation> = {
      required: true,
      min: 1,
      int: true,
      modes: {
        [FormMode.Edit]: {
          inheritShared: false,
          min: 5,
          celExpressions: [
            { celExpr: 'x >= original.x', message: 'cannot decrease' },
          ],
        },
      },
    };

    const resolved = resolveValidationForMode(v, FormMode.Edit);
    expect(resolved).toEqual({
      min: 5,
      celExpressions: [
        { celExpr: 'x >= original.x', message: 'cannot decrease' },
      ],
    });
    // base required and int should NOT be present
    expect((resolved as NumberValidation)?.required).toBeUndefined();
    expect((resolved as NumberValidation)?.int).toBeUndefined();
  });

  it('returns undefined when no base props and mode branch does not exist', () => {
    const v: ModeAwareValidation<NumberValidation> = {
      modes: {
        [FormMode.Edit]: { min: 5 },
      },
    } as ModeAwareValidation<NumberValidation>;
    expect(resolveValidationForMode(v, FormMode.New)).toBeUndefined();
  });

  it('returns mode-only rules when base is empty but mode branch exists', () => {
    const v: ModeAwareValidation<NumberValidation> = {
      modes: {
        [FormMode.Edit]: { min: 5, required: true },
      },
    } as ModeAwareValidation<NumberValidation>;
    const resolved = resolveValidationForMode(v, FormMode.Edit);
    expect(resolved).toEqual({ min: 5, required: true });
  });

  it('mode branch without celExpressions preserves base celExpressions', () => {
    const v: ModeAwareValidation<NumberValidation> = {
      min: 1,
      celExpressions: [{ celExpr: 'x > 0', message: 'positive' }],
      modes: {
        [FormMode.Edit]: { min: 5 },
      },
    };

    const resolved = resolveValidationForMode(v, FormMode.Edit);
    expect(resolved?.celExpressions).toHaveLength(1);
    expect(resolved?.min).toBe(5);
  });
});
