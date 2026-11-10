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

import { ConditionStatus, MonitoringMode } from './monitoring-card.types';

export const Messages = {
  title: 'Monitoring',
  edit: 'Edit monitoring',
  notConfigured: 'Not configured',
  destinations: 'Destinations',
  status: 'Status',
  mode: 'Mode',
  details: 'Details',
  configured: {
    True: 'Configured',
    False: 'Not configured',
    Unknown: 'Pending',
  } satisfies Record<ConditionStatus, string>,
  modes: {
    ProviderManaged: 'Provider-managed',
    ExtensionManaged: 'Extension-managed',
  } satisfies Record<MonitoringMode, string>,
};
