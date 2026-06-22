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

import type { Component } from '../../ui-generator.types';
import type { ComponentWithDataSource } from './data-source-field.types';

type OptionValue = { value: string };

export const getReconciledDataSourceValue = (
  current: unknown,
  options: OptionValue[]
): string | null => {
  const validValues = options.map((o) => o.value);
  const currentStr = typeof current === 'string' ? current : '';

  if (currentStr && !validValues.includes(currentStr)) {
    return options.length > 0 ? options[0].value : '';
  }

  if (
    (currentStr === '' || current === undefined || current === null) &&
    options.length > 0
  ) {
    return options[0].value;
  }

  return null;
};

export const hasDataSource = (
  item: Component
): item is ComponentWithDataSource => {
  return (
    'dataSource' in item &&
    typeof (item as Record<string, unknown>).dataSource === 'object' &&
    !!(item as Record<string, unknown>).dataSource
  );
};
