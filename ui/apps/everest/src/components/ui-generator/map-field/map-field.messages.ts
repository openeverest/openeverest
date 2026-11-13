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

const FIELD_NAMES: Partial<Record<string, string>> = {
  key: 'Key',
  value: 'Value',
};

export const Messages = {
  add: 'Add',
  remove: (row: number) => `Remove ${row}`,
  // A row field without a label is named after its key in the row.
  fieldName: (key: string) => FIELD_NAMES[key] ?? key,
};
