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
// The `/clusters/main/providers` endpoint is intentionally NOT mocked: it is
// served by the real backend from the installed operators and drives the
// UIGenerator uiSchema for the overview/edit/wizard schema steps.

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

// GET clusters/main/namespaces -> NamespaceList (string[])
export const mockNamespacesRoute = (page: Page) =>
  page.route(/\/namespaces$/, (route) => route.fulfill(json(mockNamespaces)));

// GET clusters/main/namespaces/{ns}/monitoring-configs -> MonitoringConfigList
export const mockMonitoringConfigsRoute = (page: Page, payload = mockMonitoringConfigs) =>
  page.route(/\/monitoring-configs$/, (route) =>
    route.fulfill(json(payload))
  );

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

// --- Aggregate -------------------------------------------------------------

// Registers the full default deterministic state. Order matters: the more
// specific `instances/inst-u3y` and `instances/{n}/(backups|restores|connection)`
// patterns are distinct from the `namespaces/{ns}/instances` list pattern, so
// there is no ambiguity between them.
export const installVisualMocks = async (page: Page) => {
  await mockNamespacesRoute(page);
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
