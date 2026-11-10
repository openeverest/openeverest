// everest
// Copyright (C) 2023 Percona LLC
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

// Drawn from crypto.getRandomValues: these suffixes end up in resource names,
// Secret names included. Not crypto.randomUUID, which only exists in secure
// contexts and the UI may be served over plain HTTP.
export const generateShortUID = (): string => {
  const [n] = crypto.getRandomValues(new Uint32Array(1));
  return n.toString(36).padStart(3, '0').slice(-3);
};
