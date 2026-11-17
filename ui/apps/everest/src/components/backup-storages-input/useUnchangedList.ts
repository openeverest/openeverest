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

import { useMemo } from 'react';

// The list keeps its identity while it holds the same items, so an effect
// that depends on it runs when the list changes and not on every render.
export const useUnchangedList = <T>(list: T[]): T[] => {
  const content = JSON.stringify(list);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => list, [content]);
};
