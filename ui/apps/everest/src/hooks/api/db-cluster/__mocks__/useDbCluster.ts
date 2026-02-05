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

import { ProxyExposeType } from 'shared-types/dbCluster.types.ts';
import { DbEngineType } from 'shared-types/dbEngines.types.ts';

export const useDbCluster = () => ({
  data: [
    {
      apiVersion: 'everest.percona.com/v1alpha1',
      kind: 'DatabaseCluster',
      metadata: {
        name: 'mysql-mkg',
        namespace: 'the-dark-side',
      },
      spec: {
        allowUnsafeConfiguration: true,
        monitoring: {},
        engine: {
          config:
            "[mysqld]\ninnodb_adaptive_hash_index = True\ninnodb_log_files_in_group = 2\ninnodb_parallel_read_threads = 1\ninnodb_buffer_pool_instances = 1\ninnodb_flush_method = O_DIRECT\ninnodb_log_file_size = 152548048\ninnodb_page_cleaners = 1\ninnodb_purge_threads = 4\ninnodb_buffer_pool_chunk_size = 2097152\ninnodb_buffer_pool_size = 731424661\ninnodb_ddl_threads = 2\ninnodb_flush_log_at_trx_commit = 2\ninnodb_io_capacity_max = 1800\ninnodb_monitor_enable = ALL\nreplica_parallel_workers = 4\nreplica_preserve_commit_order = ON\nreplica_compressed_protocol = 1\nreplica_exec_mode = STRICT\nreplica_parallel_type = LOGICAL_CLOCK\nloose_group_replication_member_expel_timeout = 6\nloose_group_replication_autorejoin_tries = 3\nloose_group_replication_message_cache_size = 162538812\nloose_group_replication_communication_max_message_size = 10485760\nloose_group_replication_unreachable_majority_timeout = 1029\nloose_group_replication_poll_spin_loops = 20000\nloose_group_replication_paxos_single_leader = ON\nloose_binlog_transaction_dependency_tracking = WRITESET\nread_rnd_buffer_size = 393216\nsort_buffer_size = 524288\nmax_heap_table_size = 16777216\ntmp_table_size = 16777216\nbinlog_cache_size = 131072\nbinlog_stmt_cache_size = 131072\njoin_buffer_size = 524288\nmax_connections = 72\ntablespace_definition_cache = 512\nthread_cache_size = 9\nthread_stack = 1024\ntable_open_cache_instances = 4\nsync_binlog = 1\nsql_mode = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION,TRADITIONAL,STRICT_ALL_TABLES'\nbinlog_expire_logs_seconds = 604800\nthread_pool_size = 4\ntable_definition_cache = 4096\ntable_open_cache = 4096\nbinlog_format = ROW\n    ",
          replicas: 1,
          resources: {
            cpu: 1,
            memory: '2G',
          },
          storage: {
            class: 'local-path',
            size: '25G',
          },
          type: DbEngineType.PXC,
          version: '8.0.31-23.2',
        },
        proxy: {
          expose: {
            type: ProxyExposeType.ClusterIP,
          },
          replicas: 1,
        },
      },
    },
  ],
});
