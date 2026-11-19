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

import { InstanceTone } from 'shared-types/instance.types';

// The order status words are offered in by the status filter: the most urgent
// first.
export const TONES_BY_URGENCY: readonly InstanceTone[] = [
  'error',
  'attention',
  'unknown',
  'progress',
  'stopped',
  'deleting',
  'ok',
];

// The Instances someone has to look at: the ones in trouble, and the ones no
// provider answers for.
export const NEEDS_ATTENTION_TONES: readonly InstanceTone[] = [
  'error',
  'attention',
  'unknown',
];
