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

import { api } from './api';
import { MonitoringClassList } from 'shared-types/monitoring-destinations.types';

// The provider filter returns only the classes an Instance of that provider
// can use (its status.providers entry carries a mode).
export const getMonitoringClassesFn = async (
  cluster: string,
  provider?: string
) => {
  const response = await api.get<MonitoringClassList>(
    `clusters/${cluster}/monitoring-classes`,
    { params: provider ? { provider } : undefined }
  );
  return response.data;
};
