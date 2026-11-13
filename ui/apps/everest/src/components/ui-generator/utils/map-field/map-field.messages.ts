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

import { pluralize } from '@percona/utils';

export const Messages = {
  key: 'Key',
  value: 'Value',
  keyRequired: 'Enter a key',
  duplicateKey: (key: string) => `${key} is set more than once`,
  required: 'Add at least one entry',
  // Each says what to do, then the rule, since the user may be past it.
  minItems: (min: number, missing: number) =>
    `Add ${pluralize(missing, 'more entry', 'more entries')} (at least ${min} required)`,
  maxItems: (max: number, extra: number) =>
    `Remove ${pluralize(extra, 'entry', 'entries')} (at most ${max} allowed)`,
};
