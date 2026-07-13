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

// Playwright network-layer mocks for the visual-regression suite.
//
// All handlers fulfill with fixed JSON fixtures so the UI renders
// deterministically. Real requests go through an axios instance with a base
// path, so URLs look like `/v1/clusters/main/namespaces/default/instances/...`.
// We match them with tail regexes (proven by the previous committed baselines).
//
// The `/clusters/main/providers` endpoint is mocked from a captured fixture
// (see ./providers.data.ts) so the suite is fully hermetic and does not depend
// on the real backend having its provider operators Ready. The provider
// `spec.uiSchema` drives the UIGenerator for the overview/edit/wizard steps.

import { Page } from '@playwright/test';
import {
  mockInstance,
  mockInstanceWithSchedules,
  mockInstancesList,
  mockInstancesListEmpty,
  mockInstanceConnection,
  mockNamespaces,
  mockMonitoringConfigs,
  mockMonitoringConfigsEmpty,
  mockBackupStorages,
  mockBackups,
  mockBackupClasses,
  mockRestores,
  mockDbEngines,
} from './data';
import { mockProviders } from './providers.data';
import {
  mockAuthTokenResponse,
  mockVersion,
  mockSettings,
  mockPermissions,
  mockClusterInfo,
  mockPlugins,
} from './platform.data';

const json = (payload: unknown) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify(payload),
});

// --- Individual handlers ---------------------------------------------------

// GET clusters/main/namespaces/{ns}/instances -> InstanceList
export const mockInstancesListRoute = (
  page: Page,
  payload = mockInstancesList
) =>
  page.route(/\/namespaces\/[^/]+\/instances$/, (route) =>
    route.fulfill(json(payload))
  );

// GET clusters/main/namespaces/{ns}/instances -> empty InstanceList
export const mockInstancesListEmptyRoute = (page: Page) =>
  mockInstancesListRoute(page, mockInstancesListEmpty);

// GET clusters/main/namespaces/{ns}/instances/inst-u3y -> Instance
export const mockInstanceDetail = (
  page: Page,
  payload: unknown = mockInstance
) =>
  page.route(/\/instances\/inst-u3y$/, (route) => route.fulfill(json(payload)));

// Variant: instance detail carrying backup schedules.
export const mockInstanceDetailWithSchedules = (page: Page) =>
  mockInstanceDetail(page, mockInstanceWithSchedules);

// GET clusters/main/namespaces/{ns}/instances/{n}/connection -> InstanceConnectionDetails
export const mockInstanceConnectionRoute = (page: Page) =>
  page.route(/\/instances\/[^/]+\/connection$/, (route) =>
    route.fulfill(json(mockInstanceConnection))
  );

// GET clusters/main/namespaces -> NamespaceList (string[]).
// Anchored to the API path so it does NOT clobber the SPA route
// `/settings/namespaces` (which also ends in `/namespaces`).
export const mockNamespacesRoute = (page: Page) =>
  page.route(/\/clusters\/[^/]+\/namespaces$/, (route) =>
    route.fulfill(json(mockNamespaces))
  );

// GET clusters/main/namespaces/{ns}/monitoring-configs -> MonitoringConfigList
export const mockMonitoringConfigsRoute = (
  page: Page,
  payload = mockMonitoringConfigs
) =>
  page.route(/\/monitoring-configs$/, (route) => route.fulfill(json(payload)));

// Variant: monitoring-configs with no endpoints (no-endpoint-available state).
export const mockMonitoringConfigsEmptyRoute = (page: Page) =>
  mockMonitoringConfigsRoute(page, mockMonitoringConfigsEmpty);

// GET clusters/main/namespaces/{ns}/backup-storages -> BackupStorageList
export const mockBackupStoragesRoute = (page: Page) =>
  page.route(/\/backup-storages$/, (route) =>
    route.fulfill(json(mockBackupStorages))
  );

// GET clusters/main/namespaces/{ns}/instances/{n}/backups -> BackupList
export const mockBackupsRoute = (page: Page) =>
  page.route(/\/instances\/[^/]+\/backups$/, (route) =>
    route.fulfill(json(mockBackups))
  );

// GET clusters/main/backup-classes -> ListBackupClassesPayload
export const mockBackupClassesRoute = (page: Page) =>
  page.route(/\/backup-classes$/, (route) =>
    route.fulfill(json(mockBackupClasses))
  );

// GET clusters/main/namespaces/{ns}/instances/{n}/restores -> GetRestorePayload
export const mockRestoresRoute = (page: Page) =>
  page.route(/\/instances\/[^/]+\/restores$/, (route) =>
    route.fulfill(json(mockRestores))
  );

// GET /namespaces/{ns}/database-engines -> GetDbEnginesPayload
export const mockDbEnginesRoute = (page: Page) =>
  page.route(/\/database-engines$/, (route) =>
    route.fulfill(json(mockDbEngines))
  );

// GET clusters/main/providers -> ProviderList (drives the UIGenerator uiSchema)
export const mockProvidersRoute = (page: Page) =>
  page.route(/\/clusters\/[^/]+\/providers$/, (route) =>
    route.fulfill(json(mockProviders))
  );

// --- Platform / auth handlers ----------------------------------------------

// POST /v1/auth/token -> AuthTokenResponse. Grants a fake in-memory session so
// the app's silent bootstrap (refreshSession) authenticates without a backend.
export const mockAuthenticatedSessionRoute = (page: Page) =>
  page.route(/\/auth\/token$/, (route) =>
    route.fulfill(json(mockAuthTokenResponse))
  );

// POST /v1/auth/token -> 401. Keeps the app on the unauthenticated login page.
export const mockUnauthenticatedSessionRoute = (page: Page) =>
  page.route(/\/auth\/token$/, (route) =>
    route.fulfill({ status: 401, contentType: 'application/json', body: '{}' })
  );

// GET /v1/version
export const mockVersionRoute = (page: Page) =>
  page.route(/\/version$/, (route) => route.fulfill(json(mockVersion)));

// GET /v1/settings (empty oidcConfig -> manual login form)
export const mockSettingsRoute = (page: Page) =>
  page.route(/\/settings$/, (route) => route.fulfill(json(mockSettings)));

// GET /v1/permissions (RBAC disabled)
export const mockPermissionsRoute = (page: Page) =>
  page.route(/\/permissions$/, (route) => route.fulfill(json(mockPermissions)));

// GET /v1/cluster-info
export const mockClusterInfoRoute = (page: Page) =>
  page.route(/\/cluster-info$/, (route) =>
    route.fulfill(json(mockClusterInfo))
  );

// GET /v1/plugins (may carry a query string)
export const mockPluginsRoute = (page: Page) =>
  page.route(/\/plugins(\?|$)/, (route) => route.fulfill(json(mockPlugins)));

// Platform endpoints needed by every page (auth-independent).
export const installPlatformMocks = async (page: Page) => {
  await mockVersionRoute(page);
  await mockSettingsRoute(page);
  await mockPermissionsRoute(page);
  await mockClusterInfoRoute(page);
  await mockPluginsRoute(page);
};

// --- Aggregate -------------------------------------------------------------

// Registers the full default deterministic state. Order matters: the more
// specific `instances/inst-u3y` and `instances/{n}/(backups|restores|connection)`
// patterns are distinct from the `namespaces/{ns}/instances` list pattern, so
// there is no ambiguity between them.
export const installVisualMocks = async (page: Page) => {
  await mockNamespacesRoute(page);
  await mockProvidersRoute(page);
  await mockDbEnginesRoute(page);
  await mockInstancesListRoute(page);
  await mockInstanceDetail(page);
  await mockInstanceConnectionRoute(page);
  await mockMonitoringConfigsRoute(page);
  await mockBackupStoragesRoute(page);
  await mockBackupsRoute(page);
  await mockBackupClassesRoute(page);
  await mockRestoresRoute(page);
};
