// everest
// Copyright (C) 2023 Percona LLC
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
import { ProxyType } from '@percona/types';
import { DbEngineType } from './dbEngines.types';
import { DbErrorType } from './dbErrors.types';

export enum ProxyExposeType {
  ClusterIP = 'ClusterIP',
  LoadBalancer = 'LoadBalancer',
  NodePort = 'NodePort',
}

export enum DbClusterStatus {
  initializing = 'initializing',
  paused = 'paused',
  pausing = 'pausing',
  stopping = 'stopping',
  ready = 'ready',
  error = 'error',
  restoring = 'restoring',
  deleting = 'deleting',
  resizingVolumes = 'resizingVolumes',
  creating = 'creating',
  upgrading = 'upgrading',
  importing = 'importing',
}

export interface Schedule {
  enabled: boolean;
  name: string;
  backupStorageName: string;
  retentionCopies?: number;
  schedule: string;
}

export type ManageableSchedules = Schedule & {
  canBeManaged: boolean;
};

export interface PITR {
  enabled: boolean;
  backupStorageName: string;
}

export interface Backup {
  pitr?: PITR;
  schedules?: Array<Schedule>;
}

export interface Resources {
  cpu: number | string;
  memory: number | string;
}

interface Storage {
  class?: string;
  size: number | string;
}

interface Engine {
  replicas: number;
  resources?: Resources;
  storage: Storage;
  crVersion?: string;
  type: DbEngineType;
  version?: string;
  config?: string;
}

export interface ProxyExposeConfig {
  type: ProxyExposeType;
  ipSourceRanges?: string[];
  loadBalancerConfigName?: string;
}

export interface Proxy {
  replicas?: number;
  expose: ProxyExposeConfig;
  resources?: Resources;
  type: ProxyType;
}

export interface DataImporter {
  dataImporterName: string;
  source: {
    path: string;
    s3: {
      accessKeyId: string;
      bucket: string;
      credentialsSecretName: string;
      endpointURL: string;
      region: string;
      secretAccessKey: string;
      verifyTLS: boolean;
    };
  };
}

export interface DataSource {
  dataImport?: DataImporter;
  dbClusterBackupName?: string;
  pitr?: DataSourcePitr;
}

export interface DataSourcePitr {
  date: string;
  type: 'date';
}

export interface Monitoring {
  monitoringConfigName?: string;
}

export interface Sharding {
  configServer: {
    replicas: number;
  };
  shards: number;
  enabled: boolean;
}

export interface EngineFeatures {
  psmdb?: {
    splitHorizonDnsConfigName?: string;
  };
}

export interface StatusEngineFeatures {
  psmdb?: {
    splitHorizon?: {
      domains?: {
        domain?: string;
        privateIP?: string;
        publicIP?: string;
      }[];
      host?: string;
    };
  };
}

export interface Spec {
  allowUnsafeConfiguration?: boolean;
  backup?: Backup;
  engine: Engine;
  proxy: Proxy;
  paused?: boolean;
  dataSource?: DataSource;
  monitoring: Monitoring;
  sharding?: Sharding;
  podSchedulingPolicyName?: string;
  engineFeatures?: EngineFeatures;
}
export interface StatusCondition {
  type: DbErrorType;
  status: string;
  observedGeneration: number;
  lastTransitionTime: string;
  reason: string;
  message: string;
}

export interface StatusSpec {
  status: DbClusterStatus;
  hostname: string;
  port: number;
  activeStorage?: string;
  crVersion: string;
  recommendedCRVersion?: string;
  details?: string;
  conditions: StatusCondition[];
  engineFeatures?: StatusEngineFeatures;
}

export interface DbClusterMetadata {
  generation?: number;
  resourceVersion?: string;
  name: string;
  namespace: string;
  annotations?: {
    'everest.percona.com/restart'?: string;
  };
}

export interface DbCluster {
  apiVersion: string;
  kind: 'DatabaseCluster';
  metadata: DbClusterMetadata;
  spec: Spec;
  status?: StatusSpec;
}

export type GetDbClusterPayload = {
  apiVersion: string;
  items: Array<DbCluster>;
  kind: 'DatabaseCluster';
  metadata: DbClusterMetadata;
};

export type ClusterCredentials = {
  username: string;
  password: string;
  connectionUrl?: string;
};

export type GetDbClusterCredentialsPayload = ClusterCredentials;
