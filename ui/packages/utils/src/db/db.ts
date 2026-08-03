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
