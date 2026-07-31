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

import { PitrStorageStatus, PitrWindow } from './restore-pitr.types';

// Serialize a picked moment to RFC 3339 with an explicit UTC offset. The BE
// rejects timezone-less timestamps (several engines read them as node-local),
// and toISOString always yields UTC 'Z'; millis are dropped for a clean value.
export const toRestoreDateISO = (date: Date): string =>
  `${date.toISOString().split('.')[0]}Z`;

// Resolve the per-storage PITR status into a picker-ready window. The window is
// only usable when the provider reports it as Available with both bounds set;
// otherwise the storage cannot be restored to a point in time and the human
// message explains why.
export const getPitrWindow = (
  status: PitrStorageStatus | undefined
): PitrWindow => {
  if (
    status?.state !== 'Available' ||
    !status.earliestRestorableTime ||
    !status.latestRestorableTime
  ) {
    return { available: false, message: status?.message };
  }

  return {
    available: true,
    earliest: new Date(status.earliestRestorableTime),
    latest: new Date(status.latestRestorableTime),
    message: status.message,
  };
};
