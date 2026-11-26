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

const warnedRules = new Set<string>();

// The form only says a rule couldn't be evaluated; the rule and the reason go
// here for the provider author. Validation reruns on every change, so each
// rule is reported once.
export const warnRuleNotEvaluable = (
  fieldPath: string,
  celExpr: string,
  evaluationError: string
): void => {
  const rule = `${fieldPath}\n${celExpr}`;
  if (warnedRules.has(rule)) return;
  warnedRules.add(rule);

  // eslint-disable-next-line no-console
  console.warn(
    `[UISchema] The CEL rule on "${fieldPath}" could not be evaluated: ${celExpr}\n${evaluationError}`
  );
};
