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

import { isPlainObject } from 'components/ui-generator/utils/object-path';

const toCanonicalJson = (value: unknown): unknown => {
  if (Array.isArray(value)) {
    return value.map(toCanonicalJson);
  }
  if (!isPlainObject(value)) {
    return value;
  }
  return Object.fromEntries(
    Object.keys(value)
      .sort()
      .map((key) => [key, toCanonicalJson(value[key])])
  );
};

// As the server compares a stored and an updated spec: as JSON documents,
// whatever the order of their keys.
export const isSameJsonDocument = (a: unknown, b: unknown) =>
  JSON.stringify(toCanonicalJson(a)) === JSON.stringify(toCanonicalJson(b));
