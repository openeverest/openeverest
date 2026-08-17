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

import { PitrStorageStatus, PitrWindow } from 'shared-types/pitr.types';

// Canonical resolver shared by the restore modal (per-storage option) and the
// cluster-details storages panel (row caption), so both surfaces parse the
// provider window identically. The "usable" rule is documented on PitrWindow.
export const resolvePitrWindow = (
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
