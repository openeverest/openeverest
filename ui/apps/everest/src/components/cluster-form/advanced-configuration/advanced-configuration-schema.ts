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

import { z } from 'zod';
import { AdvancedConfigurationFields } from './advanced-configuration.types';
import { ProxyExposeType } from 'shared-types/dbCluster.types';
import { IP_REGEX } from 'consts';
import { Messages } from './messages';

export const advancedConfigurationsSchema = () =>
  z
    .object({
      [AdvancedConfigurationFields.storageClass]: z
        .string()
        .nullable()
        .superRefine((input, ctx) => {
          if (!input) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: Messages.errors.storageClass.invalid,
            });
          }
        }),
      [AdvancedConfigurationFields.sourceRanges]: z.array(
        z.object({ sourceRange: z.string().optional() })
      ),
      [AdvancedConfigurationFields.engineParametersEnabled]: z.boolean(),
      [AdvancedConfigurationFields.engineParameters]: z.string().optional(),
      [AdvancedConfigurationFields.podSchedulingPolicyEnabled]: z.boolean(),
      [AdvancedConfigurationFields.podSchedulingPolicy]: z.string().optional(),
      [AdvancedConfigurationFields.loadBalancerConfigName]: z
        .string()
        .optional(),
      [AdvancedConfigurationFields.exposureMethod]:
        z.nativeEnum(ProxyExposeType),
      [AdvancedConfigurationFields.splitHorizonDNSEnabled]: z.boolean(),
      [AdvancedConfigurationFields.splitHorizonDNS]: z.string().optional(),
    })
    .passthrough()
    .superRefine(({ sourceRanges, exposureMethod }, ctx) => {
      if (exposureMethod !== ProxyExposeType.LoadBalancer) {
        return;
      }

      const nonEmptyRanges = sourceRanges
        .map(({ sourceRange }) => sourceRange)
        .filter((range): range is string => !!range);

      // Format: { [range]: [indexes] }
      const duplicateIndexes: Record<string, number[]> = {};
      sourceRanges.forEach(({ sourceRange }, index) => {
        if (sourceRange) {
          // Validate if it's a valid IP using regex
          if (IP_REGEX.exec(sourceRange) === null) {
            ctx.addIssue({
              code: z.ZodIssueCode.invalid_string,
              validation: 'ip',
              path: [
                AdvancedConfigurationFields.sourceRanges,
                index,
                'sourceRange',
              ],
              message: Messages.errors.sourceRange.invalid,
            });
          }

          const duplicateEntry = nonEmptyRanges.filter(
            (item) => item === sourceRange
          );

          // We exclude our current entry
          if (duplicateEntry.length > 1) {
            duplicateIndexes[sourceRange] = duplicateIndexes[sourceRange] || [];
            duplicateIndexes[sourceRange].push(index);
          }
        }
      });

      // Check for duplicates
      Object.entries(duplicateIndexes).forEach(([, indexes]) => {
        if (indexes.length >= 2) {
          // Remove the first index to avoid adding an issue for the first occurrence
          indexes.slice(1).forEach((index) => {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: [
                AdvancedConfigurationFields.sourceRanges,
                index,
                'sourceRange',
              ],
              message: Messages.errors.sourceRange.duplicate,
            });
          });
        }
      });
    });

export type AdvancedConfigurationFormType = z.infer<
  ReturnType<typeof advancedConfigurationsSchema>
>;
