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

import { z } from 'zod';
import {
  AffinityOperator,
  AffinityPriority,
  AffinityType,
  NUMERIC_AFFINITY_OPERATORS,
} from 'shared-types/affinity.types';
import { doesAffinityOperatorRequireValues } from 'utils/db';
import { PerconaZodCustomIssue } from 'utils/common-validation';
import { AffinityFormFields } from 'pages/settings/policies/pod-scheduling-policies/affinity/affinity-form-dialog/affinity-form/affinity-form.types';
import { Messages } from './group-editor-dialog/group-editor-dialog.messages';

const CONDITIONS = 'conditions';
const PASSTHROUGH = 'passthrough';

// k8s label value: up to 63 chars, alphanumerics plus - _ ., must start and end
// with an alphanumeric.
const LABEL_VALUE_PATTERN = /^[A-Za-z0-9]([A-Za-z0-9._-]{0,61}[A-Za-z0-9])?$/;
const INTEGER_PATTERN = /^-?\d+$/;

const conditionSchema = z.object({
  [AffinityFormFields.key]: z.string().optional(),
  [AffinityFormFields.operator]: z.nativeEnum(AffinityOperator).optional(),
  [AffinityFormFields.values]: z.array(z.string()).optional(),
});

const groupBaseSchema = z.object({
  [AffinityFormFields.type]: z.nativeEnum(AffinityType),
  [AffinityFormFields.priority]: z.nativeEnum(AffinityPriority),
  [AffinityFormFields.weight]: z
    .union([z.number(), z.string().transform((s) => parseInt(s, 10))])
    .optional(),
  [AffinityFormFields.topologyKey]: z.string().optional(),
  [CONDITIONS]: z.array(conditionSchema),
  [PASSTHROUGH]: z.record(z.unknown()).optional(),
});

const conditionRequiredIssue = (field: string, index: number) => ({
  ...PerconaZodCustomIssue.required(field),
  path: [CONDITIONS, index, field],
});

export const affinityGroupSchema = groupBaseSchema.superRefine((group, ctx) => {
  const { type, priority, weight, topologyKey, conditions, passthrough } =
    group;

  if (
    priority === AffinityPriority.Preferred &&
    (weight === undefined || !(weight >= 1 && weight <= 100))
  ) {
    ctx.addIssue(
      PerconaZodCustomIssue.between(AffinityFormFields.weight, 1, 100)
    );
  }

  if (type !== AffinityType.NodeAffinity && !topologyKey) {
    ctx.addIssue(
      PerconaZodCustomIssue.required(
        AffinityFormFields.topologyKey,
        Messages.topologyKeyLabel
      )
    );
  }

  // A term kept from outside the UI (e.g. matchFields only) may have no
  // conditions the editor models.
  if (conditions.length === 0 && !passthrough) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: Messages.conditionsRequired,
      path: [CONDITIONS],
    });
  }

  conditions.forEach((condition, index) => {
    const key = condition[AffinityFormFields.key];
    const operator = condition[AffinityFormFields.operator];
    const values = condition[AffinityFormFields.values];

    // Every condition needs a key and an operator: a key-less pod condition
    // would leave the term without a label selector, which matches no pods.
    // Values are required only for operators that consume them.
    if (!key) {
      ctx.addIssue(conditionRequiredIssue(AffinityFormFields.key, index));
    }

    if (!operator) {
      ctx.addIssue(conditionRequiredIssue(AffinityFormFields.operator, index));
    }

    if (
      operator &&
      doesAffinityOperatorRequireValues(operator) &&
      (!values || values.length === 0)
    ) {
      ctx.addIssue(conditionRequiredIssue(AffinityFormFields.values, index));
      return;
    }

    if (operator && NUMERIC_AFFINITY_OPERATORS.includes(operator)) {
      if (type !== AffinityType.NodeAffinity) {
        ctx.addIssue({
          ...conditionRequiredIssue(AffinityFormFields.operator, index),
          message: Messages.numericOperatorNodeOnly,
        });
      }
      if (values?.length !== 1 || !INTEGER_PATTERN.test(values[0])) {
        ctx.addIssue({
          ...conditionRequiredIssue(AffinityFormFields.values, index),
          message: Messages.numericValueInvalid,
        });
      }
      return;
    }

    // Values map to k8s label values: <=63 chars, alphanumerics plus - _ .,
    // starting and ending alphanumeric. Flag the field with a clear message
    // when any entry breaks those rules.
    if (values && values.some((value) => !LABEL_VALUE_PATTERN.test(value))) {
      ctx.addIssue({
        ...conditionRequiredIssue(AffinityFormFields.values, index),
        message: Messages.invalidLabelValues,
      });
    }
  });
});
