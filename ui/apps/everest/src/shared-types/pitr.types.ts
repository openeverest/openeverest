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

import { Instance } from './api.types';

// Per-storage PITR status reported on the Instance; the recovery window and its
// trustworthiness live here.
export type PitrStorageStatus = NonNullable<
  NonNullable<
    NonNullable<NonNullable<Instance['status']>['backup']>['storages']
  >[number]['pitr']
>;

// FE-friendly recovery window resolved from the provider-reported status. The
// window is usable only when reported Available with both bounds set; otherwise
// the storage can't be restored to a point in time and the message explains why.
export interface PitrWindow {
  available: boolean;
  earliest?: Date;
  latest?: Date;
  message?: string;
}
