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

export const Messages = {
  createDatabase: 'Create database',
  statusProvider: {
    up: 'Up',
    down: 'Down',
    initializing: 'Initializing',
    pausing: 'Pausing',
    stopping: 'Stopping',
    paused: 'Paused',
    restoring: 'Restoring',
    deleting: 'Deleting',
    resizingVolumes: 'Resizing volumes',
    creating: 'Creating',
    upgrading: 'Upgrading',
    importing: 'Importing',
    importFailed: 'Import Failed',
  },
  lastBackup: {
    warningTooltip: 'Check your backups page for more info',
    inactive: 'Inactive',
    scheduled: 'Scheduled',
    pending: 'Pending',
    notStarted: 'Not Started',
    seconds: 'sec',
    minutes: 'min',
    hours: 'h',
    days: 'd',
    ago: 'ago',
  },
  expandedRow: {
    connection: 'Connection',
    dbClusterParams: 'DB Cluster parameters',
    k8sCluster: 'K8s Cluster Name',
    cpu: 'CPU',
    memory: 'Memory',
    disk: 'Disk',
    externalAccess: 'External Access',
    monitoring: 'Monitoring',
    enabled: 'Enabled',
    disabled: 'Disabled',
    shardingStatus: 'Status',
    sharding: 'Sharding',
    nrOfShards: 'Nº shards',
    configServers: 'Config servers',
  },
  dbCluster: {
    noData:
      'You currently do not have any database cluster. Create one to get started.',
  },
  responseMessages: {
    restart: 'The database is being restarted',
    resume: 'The database is being resumed',
    pause: 'The database is being suspended',
  },
};
