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

export const Messages = {
  loading: 'Loading...',
  loadFailed: 'Failed to load options',
  noOptions: 'No options available',
  unavailableOption: (value: string) => `${value} (unavailable)`,
  unavailableValueHint:
    "Not found in the available options — it may have been deleted or you may not have access to it. It's kept unless you pick another one.",
  unavailableLockedValueHint:
    "Not found in the available options — it may have been deleted or you may not have access to it. It's kept as is.",
};
