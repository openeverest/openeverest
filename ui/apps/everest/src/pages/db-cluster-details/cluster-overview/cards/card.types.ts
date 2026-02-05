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

import { DbType } from '@percona/types';
import { DbCluster, ProxyExposeType } from 'shared-types/dbCluster.types';

export type OverviewCardProps = {
  loading?: boolean;
};

export type BasicInformationOverviewCardProps = {
  type: DbType;
  name: string;
  namespace: string;
  version: string;
} & OverviewCardProps;

export type ConnectionDetailsOverviewCardProps = {
  loadingClusterDetails: boolean;
  // Since we do hostname.split, we must do proper checks
  hostname?: string;
  port: number;
  username: string;
  password: string;
  connectionUrl: string;
  splitHorizonUrl?: string;
  clusterName?: string;
  clusterNamespace?: string;
  type: DbType;
  exposeType?: ProxyExposeType;
} & OverviewCardProps;

export type AdvancedConfigurationOverviewCardProps = {
  externalAccess: boolean;
  parameters: boolean;
  storageClass: string;
  podSchedulingPolicy?: string;
  loadBalancerConfig?: string;
  splitHorizonDNS?: string;
  splitHorizonDomains?: {
    domain?: string;
    privateIP?: string;
    publicIP?: string;
  }[];
} & OverviewCardProps;

export type MonitoringConfigurationOverviewCardProps = {
  monitoring?: string;
} & OverviewCardProps;

export type DatabaseDetailsOverviewCardProps =
  BasicInformationOverviewCardProps &
    ConnectionDetailsOverviewCardProps &
    AdvancedConfigurationOverviewCardProps &
    MonitoringConfigurationOverviewCardProps &
    OverviewCardProps;

export type ResourcesDetailsOverviewProps = {
  dbCluster: DbCluster;
  sharding: DbCluster['spec']['sharding'];
  canUpdate: boolean;
} & OverviewCardProps;

export type BackupsDetailsOverviewCardProps = {
  schedules: NonNullable<DbCluster['spec']['backup']>['schedules'];
  pitrEnabled: NonNullable<
    NonNullable<DbCluster['spec']['backup']>['pitr']
  >['enabled'];
  pitrStorageName: NonNullable<
    NonNullable<DbCluster['spec']['backup']>['pitr']
  >['backupStorageName'];
  showStorage: boolean;
  dbClusterName: string;
  namespace: string;
} & OverviewCardProps;
