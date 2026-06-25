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
export enum SettingsTabs {
  storageLocations = 'storage-locations',
  monitoringEndpoints = 'monitoring-endpoints',
  namespaces = 'namespaces',
  // policies = 'policies',
}

export const settingsTabsMui: Record<SettingsTabs, number> = {
  [SettingsTabs.storageLocations]: 1,
  [SettingsTabs.monitoringEndpoints]: 2,
  [SettingsTabs.namespaces]: 3,
  // [SettingsTabs.policies]: 4,
};
