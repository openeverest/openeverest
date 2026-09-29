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

import { Provider } from 'shared-types/api.types';
import {
  SchedulingPolicyType,
  SchedulingSupportByComponent,
  schedulingPolicyTypes,
} from 'shared-types/podSchedulingPolicy.types';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

const supportedPolicyTypes = (
  supportedFields: { openAPIV3Schema?: unknown } | undefined
): SchedulingPolicyType[] => {
  // Unset supportedFields means the provider constrains nothing.
  if (supportedFields === undefined) {
    return [...schedulingPolicyTypes];
  }
  const schema = supportedFields.openAPIV3Schema;
  if (!isRecord(schema)) {
    return [];
  }
  const properties = isRecord(schema.properties)
    ? schema.properties
    : undefined;
  const schedulingPolicy =
    properties && isRecord(properties.schedulingPolicy)
      ? properties.schedulingPolicy
      : undefined;
  if (!schedulingPolicy) {
    return [];
  }
  const declaredFields = isRecord(schedulingPolicy.properties)
    ? schedulingPolicy.properties
    : undefined;
  // schedulingPolicy declared without narrowing honours the whole struct.
  if (!declaredFields) {
    return [...schedulingPolicyTypes];
  }
  return schedulingPolicyTypes.filter((type) => type in declaredFields);
};

export const deriveSchedulingSupport = (
  provider: Provider | undefined,
  topology: string | undefined
): SchedulingSupportByComponent => {
  const components =
    topology !== undefined
      ? provider?.spec?.topologies?.[topology]?.components
      : undefined;
  if (!components) {
    return {};
  }
  const support: SchedulingSupportByComponent = {};
  for (const [name, component] of Object.entries(components)) {
    const types = supportedPolicyTypes(component?.supportedFields);
    if (types.length > 0) {
      support[name] = types;
    }
  }
  return support;
};
