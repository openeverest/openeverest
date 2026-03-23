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

import { DbType, DbEngineType, ProxyType } from '@percona/types';

export const dbEngineToDbType = (dbEngine: DbEngineType): DbType => {
  switch (dbEngine) {
    case DbEngineType.PSMDB:
      return DbType.Mongo;
    case DbEngineType.PXC:
      return DbType.Mysql;
    default:
      return DbType.Postresql;
  }
};

export const dbTypeToDbEngine = (dbType: DbType): DbEngineType => {
  switch (dbType) {
    case DbType.Mongo:
      return DbEngineType.PSMDB;
    case DbType.Mysql:
      return DbEngineType.PXC;
    default:
      return DbEngineType.POSTGRESQL;
  }
};

// TODO It is no longer needed in v2, since the name of the provider is set by plugin developer
export const beautifyDbTypeName = (dbType: DbType): string => {
  switch (dbType) {
    case DbType.Mongo:
      return 'MongoDB';
    case DbType.Mysql:
      return 'MySQL';
    default:
      return 'PostgreSQL';
  }
};

export const dbTypeToProxyType = (dbType: DbType): ProxyType => {
  switch (dbType) {
    case DbType.Mongo:
      return 'mongos';
    case DbType.Mysql:
      return 'haproxy';
    default:
      return 'pgbouncer';
  }
};
