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

import { providerRegistry } from './registry';
import { useMonitoringConfigsOptions } from 'hooks/api/monitoring/useMonitoringConfigsOptions';
import { useStorageClassesOptions } from 'hooks/api/kubernetesClusters/useStorageClassesOptions';
import { MonitoringEmptyFallback } from './fallbacks';

providerRegistry.register('monitoringConfigs', {
  description: 'MonitoringConfig names in the current namespace.',
  useOptions: useMonitoringConfigsOptions,
  emptyStateFallback: { component: MonitoringEmptyFallback },
});

providerRegistry.register('storageClasses', {
  description: 'StorageClass names available on the cluster.',
  useOptions: useStorageClassesOptions,
  emptyStateFallback: null,
});
