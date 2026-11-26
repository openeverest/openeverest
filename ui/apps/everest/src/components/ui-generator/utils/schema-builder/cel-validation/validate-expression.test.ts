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

import { describe, expect, it } from 'vitest';
import { Messages } from './cel-validation.messages';
import { validateCelExpression } from './validate-expression';

const MESSAGE = 'Disk size cannot be decreased';
const data = { spec: { size: 10 } };

describe('validateCelExpression', () => {
  it("fails with the rule's message when the rule returns false", () => {
    expect(
      validateCelExpression(
        { celExpr: 'spec.size >= original.spec.size', message: MESSAGE },
        data,
        { spec: { size: 20 } }
      )
    ).toEqual({ isValid: false, message: MESSAGE });
  });

  it.each([
    ['reads a missing key', 'spec.storage.size >= 1'],
    ['compares mismatched types', 'spec.size >= original.spec.size'],
    ['has a syntax error', 'spec.size >='],
  ])('blames the rule, not the value, when it %s', (_case, celExpr) => {
    const result = validateCelExpression({ celExpr, message: MESSAGE }, data, {
      spec: { size: '20' },
    });

    expect(result.isValid).toBe(false);
    expect(result.message).toBe(Messages.ruleNotEvaluable);
    expect(result.evaluationError).toEqual(expect.any(String));
  });
});
