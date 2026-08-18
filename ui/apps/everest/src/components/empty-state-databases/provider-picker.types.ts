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

import type { Provider } from 'shared-types/api.types';
import type { GetProviderMeta } from 'components/provider-identity';

// Shared props for the empty-state provider pickers. EmptyStateDatabases owns
// the data (providers + catalog) and chooses which picker to render: rich tiles
// when the plugin-hub catalog is available, plain buttons otherwise. Both
// pickers are pure presentational components that receive the resolved data.
export interface ProviderPickerProps {
  providers: Provider[];
  getProviderMeta: GetProviderMeta;
  showImport?: boolean;
}
