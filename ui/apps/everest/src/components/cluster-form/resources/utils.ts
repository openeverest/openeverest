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

import { coerce } from 'semver';
import { humanizedResourceSizeMap, ResourceSize } from './constants';
import { Messages } from './messages';

export const isVersion84x = (version?: string): boolean => {
  const s = coerce(version);
  return !!s && s.major === 8 && s.minor === 4;
};

export const humanizeResourceSizeMap = (type: ResourceSize): string =>
  humanizedResourceSizeMap[type];

const estimated = (value: string | number | undefined, units: string) =>
  value ? `Estimated available: ${value} ${units}` : '';

export const checkResourceText = (
  value: string | number | undefined,
  units: string,
  fieldLabel: string,
  exceedFlag: boolean
) => {
  if (value) {
    const parsedNumber = Number(value);

    if (Number.isNaN(parsedNumber)) {
      return '';
    }

    const processedValue =
      fieldLabel === 'cpu' ? parsedNumber / 1000 : parsedNumber / 10 ** 9;

    if (exceedFlag) {
      return Messages.resourcesCapacityExceeding(
        fieldLabel,
        processedValue,
        units
      );
    }
    return estimated(processedValue.toFixed(2), units);
  }
  return '';
};
