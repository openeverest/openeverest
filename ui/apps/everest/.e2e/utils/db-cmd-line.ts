// everest
// Copyright (C) 2023 Percona LLC
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

import { execSync } from 'child_process';
import { expect } from '@playwright/test';

const delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

export const getPGStsName = async (cluster: string, namespace: string) => {
  try {
    const command = `kubectl get sts --namespace ${namespace} --selector=app.kubernetes.io/instance=${cluster},app.kubernetes.io/component=pg -o 'jsonpath={.items[*].metadata.name}'`;
    const output = execSync(command).toString();
    const resultArray = output.split(' ');
    return resultArray;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const getDBHost = async (cluster: string, namespace: string) => {
  try {
    const command = `kubectl get --namespace ${namespace} DatabaseClusters ${cluster} -ojsonpath='{.status.hostname}'`;
    const output = execSync(command).toString();
    return output;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const getDBType = async (cluster: string, namespace: string) => {
  try {
    const command = `kubectl get --namespace ${namespace} DatabaseClusters ${cluster} -ojsonpath='{.spec.engine.type}'`;
    const output = execSync(command).toString();
    return output;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const getDBClientPod = async (dbType: string, namespace: string) => {
  try {
    if (dbType === 'pxc') {
      dbType = 'mysql';
    }

    const command = `kubectl get pods --namespace ${namespace} --selector=name=${dbType}-client -o 'jsonpath={.items[].metadata.name}'`;
    const output = execSync(command).toString();
    return output;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const getPXCPassword = async (cluster: string, namespace: string) => {
  try {
    const command = `kubectl get secret --namespace ${namespace} everest-secrets-${cluster} -o template='{{ .data.root | base64decode }}'`;
    const output = execSync(command).toString();
    return output;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const getPSMDBPassword = async (cluster: string, namespace: string) => {
  try {
    const command = `kubectl get secret --namespace ${namespace} everest-secrets-${cluster} -o template='{{ .data.MONGODB_BACKUP_PASSWORD | base64decode }}'`;
    const output = execSync(command).toString();
    return output;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const getPGPassword = async (cluster: string, namespace: string) => {
  try {
    const command = `kubectl get secret --namespace ${namespace} everest-secrets-${cluster} -o template='{{ .data.password | base64decode }}'`;
    const output = execSync(command).toString();
    return output;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const getPSMDBShardingStatus = async (
  cluster: string,
  namespace: string
) => {
  try {
    const command = `kubectl get --namespace ${namespace} DatabaseClusters ${cluster} -ojsonpath='{.spec.sharding.enabled}'`;
    const output = execSync(command).toString();
    return output;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const queryMySQL = async (
  cluster: string,
  namespace: string,
  query: string,
  retry: number = 2
): Promise<string> => {
  const password = await getPXCPassword(cluster, namespace);
  const clientPod = await getDBClientPod('mysql', 'db-client');

  let attempt = 0;
  let lastError: any;

  while (attempt < retry) {
    try {
      const host = await getDBHost(cluster, namespace);
      const command = `kubectl exec --namespace db-client ${clientPod} -- mysqlsh root@${host} -p'${password}' --sql --quiet-start=2 -e '${query}' | tail -n +2`;
      const output = execSync(command).toString();
      return output;
    } catch (error) {
      lastError = error;
      attempt++;
      if (attempt < retry) {
        await delay(2000);
      }
    }
  }

  console.error(
    `Failed to execute command in queryMySQL in ${attempt} attempts: ${lastError}`
  );
  throw lastError;
};

export const queryPSMDB = async (
  cluster: string,
  namespace: string,
  db: string,
  query: string,
  retry: number = 2
): Promise<string> => {
  const password = await getPSMDBPassword(cluster, namespace);
  const clientPod = await getDBClientPod('psmdb', 'db-client');

  // Enable replicaSet option if sharding is disabled
  const isShardingEnabled =
    (await getPSMDBShardingStatus(cluster, namespace)) === 'true';
  const replicaSetOption = isShardingEnabled ? '' : '&replicaSet=rs0';

  let attempt = 0;
  let lastError: any;

  while (attempt < retry) {
    try {
      const host = await getDBHost(cluster, namespace);
      const command = `kubectl exec --namespace db-client ${clientPod} -- mongosh "mongodb://backup:${password}@${host}/${db}?authSource=admin${replicaSetOption}" --eval "${query}"`;
      const output = execSync(command).toString();
      return output;
    } catch (error) {
      lastError = error;
      attempt++;
      if (attempt < retry) {
        await delay(2000);
      }
    }
  }

  console.error(
    `Failed to execute command in queryPSMDB in ${attempt} attempts: ${lastError}`
  );
  throw lastError;
};

export const queryPG = async (
  cluster: string,
  namespace: string,
  db: string,
  query: string,
  retry: number = 2
): Promise<string> => {
  const password = await getPGPassword(cluster, namespace);
  const clientPod = await getDBClientPod('postgresql', 'db-client');

  let attempt = 0;
  let lastError: any;

  while (attempt < retry) {
    try {
      const host = await getDBHost(cluster, namespace);
      const command = `kubectl exec --namespace db-client ${clientPod} -- bash -c "PGPASSWORD='${password}' psql -h${host} -p5432 -Upostgres -d${db} -t -q -c'${query}'"`;
      const output = execSync(command).toString();
      return output;
    } catch (error) {
      lastError = error;
      attempt++;
      if (attempt < retry) {
        await delay(2000);
      }
    }
  }

  console.error(
    `Failed to execute command in queryPG in ${attempt} attempts: ${lastError}`
  );
  throw lastError;
};

export const prepareTestDB = async (cluster: string, namespace: string) => {
  const dbType = await getDBType(cluster, namespace);

  switch (dbType) {
    case 'pxc': {
      await dropTestDB(cluster, namespace);
      await queryMySQL(
        cluster,
        namespace,
        'CREATE DATABASE test; CREATE TABLE test.t1 (a INT PRIMARY KEY); INSERT INTO test.t1 VALUES (1),(2),(3);'
      );
      const result = await queryTestDB(cluster, namespace);
      expect(result.trim()).toBe('1\n2\n3');
      break;
    }
    case 'psmdb': {
      await dropTestDB(cluster, namespace);
      await queryPSMDB(
        cluster,
        namespace,
        'test',
        'db.t1.insertMany([{ a: 1 }, { a: 2 }, { a: 3 }]);'
      );
      const result = await queryTestDB(cluster, namespace);
      expect(result.trim()).toBe('[{"a":1},{"a":2},{"a":3}]');
      break;
    }
    case 'postgresql': {
      await dropTestDB(cluster, namespace);
      await queryPG(cluster, namespace, 'postgres', 'CREATE DATABASE test;');
      await queryPG(
        cluster,
        namespace,
        'test',
        'CREATE TABLE t1 (a INT PRIMARY KEY); INSERT INTO t1 VALUES (1),(2),(3);'
      );
      const result = await queryTestDB(cluster, namespace);
      expect(result.trim()).toBe('1\n 2\n 3');
      break;
    }
  }
};

export const insertTestDB = async (
  cluster: string,
  namespace: string,
  data: string[],
  expected: string[]
) => {
  const dbType = await getDBType(cluster, namespace);

  switch (dbType) {
    case 'pxc': {
      const values = data.map((d) => `(${d})`).join(',');
      await queryMySQL(
        cluster,
        namespace,
        `INSERT INTO test.t1 VALUES ${values};`
      );
      const result = await queryTestDB(cluster, namespace);
      const expected_result = expected.join('\n');
      expect(result.trim()).toBe(expected_result);
      break;
    }
    case 'psmdb': {
      const jsonData = data.map((d) => `{ a: ${d} }`).join(', ');

      await queryPSMDB(
        cluster,
        namespace,
        'test',
        `db.t1.insertMany([${jsonData}]);`
      );
      const result = await queryTestDB(cluster, namespace);
      const expected_result = expected.map((d) => `{"a":${d}}`).join(',');

      expect(result.trim()).toBe('[' + expected_result + ']');
      break;
    }
    case 'postgresql': {
      const values = data.map((d) => `(${d})`).join(',');
      await queryPG(
        cluster,
        namespace,
        'test',
        `INSERT INTO t1 VALUES ${values};`
      );
      const result = await queryTestDB(cluster, namespace);
      const expected_result = expected.join('\n');
      // Output lines contain leading spaces depending on the number of digits.
      const normalizedResult = result
        .split('\n')
        .map((line) => line.trim())
        .join('\n');
      expect(normalizedResult.trim()).toBe(expected_result);
      break;
    }
    default:
      throw new Error(`Unsupported database type: ${dbType}`);
  }
};

export const dropTestDB = async (cluster: string, namespace: string) => {
  const dbType = await getDBType(cluster, namespace);

  switch (dbType) {
    case 'pxc': {
      await queryMySQL(cluster, namespace, 'DROP DATABASE IF EXISTS test;');
      const result = await queryMySQL(cluster, namespace, 'SHOW DATABASES;');
      expect(result).not.toContain('test');
      break;
    }
    case 'psmdb': {
      await queryPSMDB(cluster, namespace, 'test', 'db.dropDatabase();');
      const result = await queryPSMDB(cluster, namespace, 'admin', 'show dbs;');
      expect(result).not.toContain('test');
      break;
    }
    case 'postgresql': {
      await queryPG(
        cluster,
        namespace,
        'postgres',
        'DROP DATABASE IF EXISTS test WITH (FORCE);'
      );
      const result = await queryPG(cluster, namespace, 'postgres', '\\list');
      expect(result).not.toContain('test');
      break;
    }
  }
};

export const queryTestDB = async (
  cluster: string,
  namespace: string,
  collection: string = 't1'
) => {
  const dbType = await getDBType(cluster, namespace);
  let result: string;

  switch (dbType) {
    case 'pxc': {
      result = await queryMySQL(cluster, namespace, 'SELECT * FROM test.t1;');
      break;
    }

    case 'psmdb': {
      result = await queryPSMDB(
        cluster,
        namespace,
        'test',
        `JSON.stringify(db.${collection}.find({},{_id: 0}).sort({a: 1}).toArray());`
      );

      break;
    }
    case 'postgresql': {
      result = await queryPG(cluster, namespace, 'test', 'SELECT * FROM t1;');
      break;
    }
  }
  return result;
};

export const prepareMongoDBTestDB = async (
  cluster: string,
  namespace: string
) => {
  await dropTestDB(cluster, namespace);

  await queryPSMDB(
    cluster,
    namespace,
    'test',
    'db.t1.insertMany([{ a: 1 }, { a: 2 }, { a: 3 }]);'
  );

  await queryPSMDB(
    cluster,
    namespace,
    'test',
    'db.t2.insertMany([{ a: 1 }, { a: 2 }, { a: 3 }]);'
  );

  await queryPSMDB(cluster, namespace, 'test', 'db.t1.createIndex({ a: 1 });');
  await queryPSMDB(cluster, namespace, 'test', 'db.t2.createIndex({ a: 1 });');

  await queryPSMDB(
    cluster,
    namespace,
    'admin',
    'sh.enableSharding(\\"test\\");'
  );
};

export const configureMongoDBSharding = async (
  cluster: string,
  namespace: string
) => {
  const shardCollection = async (
    collectionName: string,
    key: object,
    splitKey: object
  ) => {
    await queryPSMDB(
      cluster,
      namespace,
      'admin',
      `sh.shardCollection(\\"test.${collectionName}\\", ${JSON.stringify(key)});`
    );

    await queryPSMDB(
      cluster,
      namespace,
      'admin',
      `sh.splitAt(\\"test.${collectionName}\\", ${JSON.stringify(splitKey)});`
    );

    await queryPSMDB(
      cluster,
      namespace,
      'admin',
      `sh.moveChunk(\\"test.${collectionName}\\", { ${Object.keys(key)[0]} : MinKey}, \\"rs0\\");`
    );

    await queryPSMDB(
      cluster,
      namespace,
      'admin',
      `sh.moveChunk(\\"test.${collectionName}\\", { ${Object.keys(key)[0]}: MaxKey}, \\"rs1\\");`
    );
  };
  await shardCollection('t1', { a: 1 }, { a: 2 });
  await shardCollection('t2', { a: 1 }, { a: 2 });
};

export const validateMongoDBSharding = async (
  cluster: string,
  namespace: string,
  collectionName: string
) => {
  const collectionString = await queryPSMDB(
    cluster,
    namespace,
    'config',
    `db.collections.find({ _id: \\"test.${collectionName}\\" });`
  );

  const uuidMatch = collectionString.match(
    /uuid\s*:\s*UUID\(['"]([^'"]+)['"]\)/i
  );
  const collectionUUID = uuidMatch?.[1];

  const query = `db.chunks.aggregate([
    { \\$match: { uuid: UUID(\\"${collectionUUID}\\") } },
    { \\$group: { _id: \\"\\$shard\\" } } 
  ]).toArray();`;

  const queryResult = await queryPSMDB(cluster, namespace, 'config', query);

  const sanitizedResult = queryResult
    .replace(/_id/g, '"_id"')
    .replace(/'/g, '"');
  const chunks = JSON.parse(sanitizedResult);

  const shardIds = chunks.map((chunk: { _id: string }) => chunk._id);

  const hasRs0 = shardIds.includes('rs0');
  const hasRs1 = shardIds.includes('rs1');

  expect(hasRs0).toBe(true);
  expect(hasRs1).toBe(true);
};
