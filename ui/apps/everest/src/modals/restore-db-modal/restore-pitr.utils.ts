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

import { RestorePitrStorageOption } from './restore-db-modal.types';

// Serialize a picked moment to RFC 3339 with an explicit UTC offset. The BE
// rejects timezone-less timestamps (several engines read them as node-local),
// and toISOString always yields UTC 'Z'; millis are dropped for a clean value.
export const toRestoreDateISO = (date: Date): string =>
  `${date.toISOString().split('.')[0]}Z`;

// Resolve the "active" storage option: the explicitly selected one, or the first
// as a fallback for the single-storage case. Shared by the schema, the submit
// mapper, and the fields component so validation, rendering, and submission all
// agree on which storage the payload names.
export const resolveActiveStorage = (
  storages: RestorePitrStorageOption[],
  selectedName: string | undefined
): RestorePitrStorageOption | undefined =>
  storages.find((option) => option.name === selectedName) ?? storages[0];
