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

// Deterministic, pure fixtures for the visual-regression suite.
//
// NOTE ON TYPES: the src shared-types (e.g. `Instance`, `MonitoringConfigList`)
// live behind the `shared-types/*` path alias which is NOT exposed to the e2e
// tsconfig (only `@generated/api-types`, `@e2e/*`, etc. are). The generated CRD
// types also model `metadata` as an opaque `Record<string, never>`, which makes
// them impractical to satisfy from literals (see the `as unknown as` casts in
// src/hooks/api/db-instances/__mocks__/useDbInstanceList.ts). We therefore use
// plain typed object literals here and keep the shapes aligned by hand with the
// canonical sources cited in each block. This is the fallback explicitly allowed
// by the task when importing src types is awkward.

// Fixed identifiers used across the visual suite.
export const DB_NAMESPACE = 'default';
export const DB_NAME = 'inst-u3y';

// The provider name MUST match the mocked `/clusters/main/providers` fixture
// (see ./providers.data.ts) so the overview/edit UIGenerator can resolve its
// uiSchema. The provider is registered under `percona-server-mongodb` (see
// commands/instance/create.go and api/core/v1alpha1/instance_types.go). This
// intentionally differs from the `psmdb-provider` placeholder used by the src
// unit-test __mocks__.
const PROVIDER_NAME = 'percona-server-mongodb';

// Canonical shape captured verbatim from a live OpenEverest v2 instance
// (provider percona-server-mongodb, sharded topology) via
// GET /clusters/main/namespaces/{ns}/instances/{name}. The component/topology
// structure (spec.components.engine|proxy|configServer, spec.topology.type) MUST
// match what the provider uiSchema references, otherwise the schema-driven
// overview cards render empty. Only the status is synthesized (deterministic
// Ready) so screenshots are stable.
export const mockInstance = {
  apiVersion: 'core.openeverest.io/v1alpha1',
  kind: 'Instance',
  metadata: {
    name: DB_NAME,
    namespace: DB_NAMESPACE,
    creationTimestamp: '2025-10-01T00:00:00Z',
  },
  spec: {
    provider: PROVIDER_NAME,
    version: '8.0.12',
    topology: {
      type: 'sharded',
      config: { numShards: 3 },
    },
    components: {
      backupAgent: { type: 'backup', replicas: 1 },
      configServer: {
        type: 'mongod',
        storage: { size: '5Gi' },
        replicas: 3,
      },
      engine: {
        type: 'mongod',
        storage: { size: '10Gi', storageClass: 'standard' },
        resources: {
          limits: { cpu: '2', memory: '4Gi' },
          requests: { cpu: '2', memory: '4Gi' },
        },
        replicas: 3,
      },
      proxy: {
        type: 'mongos',
        replicas: 3,
        service: {
          serviceType: 'LoadBalancer',
          annotations: {
            'service.beta.kubernetes.io/aws-load-balancer-internal': 'true',
          },
        },
      },
    },
  },
  status: {
    phase: 'Ready',
    components: [
      {
        pods: [{ name: `${DB_NAME}-0` }],
        ready: 3,
        state: 'Ready',
        total: 3,
      },
    ],
    conditions: [
      {
        lastTransitionTime: '2025-10-01T00:05:00Z',
        message: 'Instance is ready',
        reason: 'Ready',
        status: 'True',
        type: 'Ready',
      },
    ],
    connectionSecretRef: { name: `${DB_NAME}-connection` },
  },
};

// A variant of the instance that carries backup schedules, used by the
// schedule-expanded / delete-schedule / schedule-form screenshots.
export const mockInstanceWithSchedules = {
  ...mockInstance,
  spec: {
    ...mockInstance.spec,
    backup: {
      enabled: true,
      classRef: { name: 'psmdb-backup-class' },
      storages: [
        {
          storageRef: { name: 's3-prod-backups' },
          schedules: [
            {
              name: 'daily-full',
              cron: '0 0 * * *',
              enabled: true,
              retentionCopies: 7,
            },
            {
              name: 'weekly-full',
              cron: '0 0 * * 0',
              enabled: true,
              retentionCopies: 4,
            },
          ],
        },
      ],
    },
  },
};

// LIST endpoint returns InstanceList = { items: Instance[] } (the select reads
// `instances.items`). See src/api/instanceApi.ts + useDbInstanceList.ts.
export const mockInstancesList = { items: [mockInstance] };

// Empty variant for the "empty state" screenshot.
export const mockInstancesListEmpty = { items: [] as (typeof mockInstance)[] };

// Connection details for the overview page (bare object).
// Shape: InstanceConnectionDetails, mirrors the src __mocks__.
export const mockInstanceConnection = {
  host: `${DB_NAME}.${DB_NAMESPACE}.svc.cluster.local`,
  port: '27017',
  username: 'admin',
  password: 'secret-password',
  uri: `mongodb://admin:secret-password@${DB_NAME}.${DB_NAMESPACE}.svc.cluster.local:27017/admin`,
  provider: PROVIDER_NAME,
  type: 'mongodb',
};

// Namespaces list: NamespaceList = string[] (see src/api/namespaces.ts +
// useNamespaces which sorts the array as strings). Must include `default`.
export const mockNamespaces = [DB_NAMESPACE];

// MonitoringConfigList = { items: MonitoringConfig[] }.
// Lifted verbatim from the proven inline fixture in visual-baseline.e2e.ts.
export const mockMonitoringConfigs = {
  apiVersion: 'monitoring.openeverest.io/v1alpha1',
  kind: 'MonitoringConfigList',
  items: [
    {
      apiVersion: 'monitoring.openeverest.io/v1alpha1',
      kind: 'MonitoringConfig',
      metadata: { name: 'pmm-prod', namespace: 'default' },
      spec: {
        type: 'pmm',
        pmm: {
          credentialsSecretRef: {
            name: 'monitoring-config-pmm-prod-credentials',
          },
          url: 'http://monitoring.example.com',
          verifyTLS: false,
        },
      },
      status: { inUse: false, lastObservedGeneration: 1 },
    },
  ],
};

// Empty variant so the create-DB wizard monitoring step renders the
// no-endpoint-available state deterministically.
export const mockMonitoringConfigsEmpty = {
  apiVersion: 'monitoring.openeverest.io/v1alpha1',
  kind: 'MonitoringConfigList',
  items: [] as (typeof mockMonitoringConfigs)['items'],
};

// BackupStorageList = { items: BackupStorage[] } (see src/api/backupStorage.ts).
// Lifted verbatim from the proven inline fixture.
export const mockBackupStorages = {
  apiVersion: 'storage.openeverest.io/v1alpha1',
  kind: 'BackupStorageList',
  items: [
    {
      apiVersion: 'storage.openeverest.io/v1alpha1',
      kind: 'BackupStorage',
      metadata: { name: 's3-prod-backups', namespace: 'default' },
      spec: {
        type: 's3',
        s3: {
          bucket: 'everest-backups-prod',
          region: 'us-east-1',
          credentialsSecretRef: {
            name: 'backup-storage-s3-prod-credentials',
          },
          endpointURL: 'https://s3.amazonaws.com',
          verifyTLS: true,
          forcePathStyle: false,
        },
      },
      status: {},
    },
  ],
};

// BackupList = { items: Backup[] } (see src/api/backups.ts getBackupsListFn).
// Lifted verbatim from the proven inline fixture.
export const mockBackups = {
  apiVersion: 'backup.openeverest.io/v1alpha1',
  kind: 'BackupList',
  items: [
    {
      apiVersion: 'backup.openeverest.io/v1alpha1',
      kind: 'Backup',
      metadata: { name: 'daily-backup-2026-06-29', namespace: 'default' },
      spec: {
        instanceName: 'inst-u3y',
        backupClassName: 'psmdb-backup-class',
        storageName: 's3-prod-backups',
        scheduleName: 'daily-full',
        deletionPolicy: 'Delete',
      },
      status: {
        state: 'Succeeded',
        startedAt: '2026-06-29T00:00:05Z',
        completedAt: '2026-06-29T00:05:30Z',
        size: '1.2Gi',
        executionMode: 'ProviderManaged',
      },
    },
    {
      apiVersion: 'backup.openeverest.io/v1alpha1',
      kind: 'Backup',
      metadata: { name: 'daily-backup-2026-06-28', namespace: 'default' },
      spec: {
        instanceName: 'inst-u3y',
        backupClassName: 'psmdb-backup-class',
        storageName: 's3-prod-backups',
        scheduleName: 'daily-full',
        deletionPolicy: 'Delete',
      },
      status: {
        state: 'Succeeded',
        startedAt: '2026-06-28T00:00:05Z',
        completedAt: '2026-06-28T00:04:12Z',
        size: '1.1Gi',
        executionMode: 'ProviderManaged',
      },
    },
    {
      apiVersion: 'backup.openeverest.io/v1alpha1',
      kind: 'Backup',
      metadata: { name: 'on-demand-backup-001', namespace: 'default' },
      spec: {
        instanceName: 'inst-u3y',
        backupClassName: 'psmdb-backup-class',
        storageName: 's3-prod-backups',
        deletionPolicy: 'Delete',
      },
      status: {
        state: 'Running',
        startedAt: '2026-06-29T10:30:00Z',
        executionMode: 'ProviderManaged',
      },
    },
  ],
};

// ListBackupClassesPayload = { items: BackupClass[] } (see src/api/backups.ts
// getBackupClassesListFn -> clusters/{cluster}/backup-classes).
// Lifted verbatim from the proven inline fixture.
export const mockBackupClasses = {
  items: [
    {
      metadata: { name: 'psmdb-backup-class' },
      spec: {
        providers: [{ name: 'percona-server-mongodb' }],
        storages: [
          { name: 's3-storage', backupStorageRef: { name: 's3-prod-backups' } },
        ],
      },
    },
  ],
};

// GetRestorePayload = { items: [...] } (see src/api/restores.ts). Empty by
// default so the restores tab renders the deterministic "no restores" state.
export const mockRestores = { items: [] as unknown[] };

// GetDbEnginesPayload for the create-DB wizard "Database Version" step.
// Raw shape (before the src `select` transforms it): availableVersions.engine is
// a MAP keyed by version string. See src/shared-types/dbEngines.types.ts and
// src/hooks/api/db-engines/useDbEngines.ts (dbEnginesQuerySelect).
export const mockDbEngines = {
  items: [
    {
      spec: { type: 'pxc' },
      status: {
        status: 'installed',
        operatorVersion: '1.15.0',
        availableVersions: {
          engine: {
            '8.0.36-28.1': {
              version: '8.0.36-28.1',
              description: '8.0.36-28.1',
              status: 'recommended',
            },
            '8.0.35-27.1': {
              version: '8.0.35-27.1',
              description: '8.0.35-27.1',
              status: 'available',
            },
          },
          backup: {},
          proxy: {},
        },
      },
      metadata: { name: 'percona-xtradb-cluster' },
    },
    {
      spec: { type: 'psmdb' },
      status: {
        status: 'installed',
        operatorVersion: '1.19.0',
        availableVersions: {
          engine: {
            '7.0.11-8': {
              version: '7.0.11-8',
              description: '7.0.11-8',
              status: 'recommended',
            },
            '6.0.19-16': {
              version: '6.0.19-16',
              description: '6.0.19-16',
              status: 'available',
            },
          },
          backup: {},
          proxy: {},
        },
      },
      metadata: { name: 'percona-server-mongodb' },
    },
    {
      spec: { type: 'postgresql' },
      status: {
        status: 'installed',
        operatorVersion: '2.5.0',
        availableVersions: {
          engine: {
            '16.4': {
              version: '16.4',
              description: '16.4',
              status: 'recommended',
            },
            '15.8': {
              version: '15.8',
              description: '15.8',
              status: 'available',
            },
          },
          backup: {},
          proxy: {},
        },
      },
      metadata: { name: 'percona-postgresql' },
    },
  ],
};
