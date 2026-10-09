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

export const RetentionType = {
  count: 'count',
  time: 'time',
  keepAll: 'keep-all',
} as const;

export type RetentionType =
  (typeof RetentionType)[keyof typeof RetentionType];

export const RetentionDurationUnit = {
  days: 'd',
  weeks: 'w',
  months: 'm',
} as const;

export type RetentionDurationUnit =
  (typeof RetentionDurationUnit)[keyof typeof RetentionDurationUnit];

export const RETENTION_DURATION_UNITS: RetentionDurationUnit[] = [
  RetentionDurationUnit.days,
  RetentionDurationUnit.weeks,
  RetentionDurationUnit.months,
];

export const DEFAULT_RETENTION_DURATION_VALUE = '30';
export const DEFAULT_RETENTION_DURATION_UNIT = RetentionDurationUnit.days;

/** Matches CRD pattern `^[1-9][0-9]*[dwm]$`. */
export const RETENTION_DURATION_PATTERN = /^([1-9][0-9]*)([dwm])$/;
