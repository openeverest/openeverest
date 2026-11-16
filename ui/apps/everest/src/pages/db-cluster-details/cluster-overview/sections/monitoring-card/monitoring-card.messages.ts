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
  ready: {
    True: 'Ready',
    False: 'Not ready',
    Unknown: 'Pending',
  } satisfies Record<ConditionStatus, string>,
  destinationTerminating: (destination: string) =>
    `${destination} is being deleted. Remove it from monitoring to let the deletion finish.`,
  destinationNotFound: (destination: string) =>
    `${destination} no longer exists. Remove it from monitoring.`,
  destinationNotReady: (destination: string) =>
    `${destination} is not ready, so no data reaches it. Its status in Settings → Monitoring destinations tells why.`,
  agentTlsNotVerified: () =>
    "The monitoring agent doesn't verify the server's TLS certificate, whatever the destination's TLS settings are.",
  integrationInUse: () =>
    'Another destination of the same monitoring system is set for this instance. Keep one of them.',
  unsupported: (destination: string) =>
    `The monitoring system of ${destination} doesn't support this instance. Remove it from monitoring.`,
  modes: {
    ProviderManaged: 'Provider-managed',
    ExtensionManaged: 'Extension-managed',
  } satisfies Record<MonitoringMode, string>,
};
