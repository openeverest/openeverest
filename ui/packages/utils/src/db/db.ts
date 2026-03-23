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
