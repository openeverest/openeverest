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

// Deterministic fixture for GET /clusters/main/providers (ProviderList).
//
// Captured verbatim from a live OpenEverest v2 cluster with the
// provider-percona-server-mongodb operator installed (chart 0.1.0), then
// stripped of volatile k8s metadata (uid, resourceVersion, creationTimestamp,
// managedFields, helm labels/annotations). The provider `spec.uiSchema` drives
// the UIGenerator for the overview/edit/wizard steps, so this must stay aligned
// with the operator version pinned in CI. Regenerate by capturing the endpoint
// response again and re-stripping metadata.
//
// The provider name (`percona-server-mongodb`) matches PROVIDER_NAME in
// ./data.ts so the mocked instance resolves against this schema.

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const mockProviders: any = {
  items: [
    {
      kind: 'Provider',
      apiVersion: 'core.openeverest.io/v1alpha1',
      metadata: {
        name: 'percona-server-mongodb',
      },
      spec: {
        componentTypes: {
          backup: {
            versions: [
              {
                version: '2.12.0',
                image: 'percona/percona-backup-mongodb:2.12.0',
                default: true,
              },
            ],
          },
          mongod: {
            versions: [
              {
                version: '6.0.19-16',
                image: 'percona/percona-server-mongodb:6.0.19-16-multi',
              },
              {
                version: '6.0.21-18',
                image: 'percona/percona-server-mongodb:6.0.21-18',
              },
              {
                version: '7.0.18-11',
                image: 'percona/percona-server-mongodb:7.0.18-11',
              },
              {
                version: '8.0.4-1',
                image: 'percona/percona-server-mongodb:8.0.4-1-multi',
              },
              {
                version: '8.0.8-3',
                image: 'percona/percona-server-mongodb:8.0.8-3',
              },
              {
                version: '8.0.12-4',
                image: 'percona/percona-server-mongodb:8.0.12-4',
                default: true,
              },
            ],
          },
          pmm: {
            versions: [
              {
                version: '2.44.1',
                image: 'percona/pmm-client:2.44.1',
              },
              {
                version: '3.7.0',
                image: 'percona/pmm-client:3.7.0',
                default: true,
              },
            ],
          },
        },
        components: {
          backupAgent: {
            type: 'backup',
          },
          configServer: {
            type: 'mongod',
          },
          engine: {
            type: 'mongod',
          },
          monitoring: {
            type: 'pmm',
          },
          proxy: {
            type: 'mongod',
          },
        },
        topologies: {
          replicaSet: {
            components: {
              backupAgent: {
                optional: true,
              },
              engine: {},
              monitoring: {
                optional: true,
              },
            },
          },
          sharded: {
            components: {
              backupAgent: {
                optional: true,
              },
              configServer: {},
              engine: {},
              monitoring: {
                optional: true,
              },
              proxy: {},
            },
            configSchema: {
              properties: {
                numShards: {
                  format: 'int32',
                  type: 'integer',
                },
              },
              type: 'object',
            },
          },
        },
        versions: [
          {
            name: '8.0.12',
            components: {
              backupAgent: '2.12.0',
              configServer: '8.0.12-4',
              engine: '8.0.12-4',
              monitoring: '3.7.0',
              proxy: '8.0.12-4',
            },
            default: true,
          },
          {
            name: '8.0.8',
            components: {
              backupAgent: '2.12.0',
              configServer: '8.0.8-3',
              engine: '8.0.8-3',
              monitoring: '3.7.0',
              proxy: '8.0.8-3',
            },
          },
          {
            name: '8.0.4',
            components: {
              backupAgent: '2.12.0',
              configServer: '8.0.4-1',
              engine: '8.0.4-1',
              monitoring: '3.7.0',
              proxy: '8.0.4-1',
            },
          },
          {
            name: '7.0.18',
            components: {
              backupAgent: '2.12.0',
              configServer: '7.0.18-11',
              engine: '7.0.18-11',
              monitoring: '3.7.0',
              proxy: '7.0.18-11',
            },
          },
          {
            name: '6.0.21',
            components: {
              backupAgent: '2.12.0',
              configServer: '6.0.21-18',
              engine: '6.0.21-18',
              monitoring: '3.7.0',
              proxy: '6.0.21-18',
            },
          },
          {
            name: '6.0.19',
            components: {
              backupAgent: '2.12.0',
              configServer: '6.0.19-16',
              engine: '6.0.19-16',
              monitoring: '3.7.0',
              proxy: '6.0.19-16',
            },
          },
        ],
        uiSchema: {
          replicaSet: {
            sections: {
              advanced: {
                components: {
                  configuration: {
                    fieldParams: {
                      label: 'Engine configuration',
                      maxRows: 8,
                      minRows: 3,
                      multiline: true,
                      placeholder:
                        'operationProfiling:\n  mode: slowOps\n  slowOpThresholdMs: 200\n',
                    },
                    path: 'spec.components.engine.configuration',
                    uiType: 'text',
                  },
                  externalAccess: {
                    fieldParams: {
                      defaultValue: 'ClusterIP',
                      label: 'External access',
                      options: [
                        {
                          label: 'ClusterIP',
                          value: 'ClusterIP',
                        },
                        {
                          label: 'LoadBalancer',
                          value: 'LoadBalancer',
                        },
                        {
                          label: 'NodePort',
                          value: 'NodePort',
                        },
                      ],
                    },
                    path: 'spec.components.engine.service.serviceType',
                    uiType: 'select',
                  },
                  storageClass: {
                    dataSource: {
                      provider: 'storageClasses',
                    },
                    fieldParams: {
                      label: 'Storage class',
                      modes: {
                        edit: {
                          disabled: true,
                        },
                        restore: {
                          disabled: true,
                        },
                      },
                    },
                    path: 'spec.components.engine.storage.storageClass',
                    uiType: 'select',
                    validation: {
                      required: true,
                    },
                  },
                },
                componentsOrder: ['storageClass', 'configuration'],
                description: 'Configure advanced settings for your database',
                label: 'Advanced configuration',
              },
              databaseVersion: {
                components: {
                  version: {
                    fieldParams: {
                      label: 'Database Version',
                      modes: {
                        edit: {
                          disabled: true,
                        },
                        restore: {
                          disabled: true,
                        },
                      },
                      optionsPath: 'spec.versions',
                      optionsPathConfig: {
                        labelPath: 'name',
                        valuePath: 'name',
                      },
                    },
                    path: 'spec.version',
                    uiType: 'select',
                    validation: {
                      required: true,
                    },
                  },
                },
                description:
                  'Provide the information about the database version you want to use.',
                label: 'Database Version',
              },
              monitoring: {
                components: {
                  monitoring: {
                    dataSource: {
                      provider: 'monitoringConfigs',
                    },
                    fieldParams: {
                      label: 'Monitoring endpoint',
                    },
                    path: 'spec.components.monitoring.customSpec.monitoringConfigName',
                    uiType: 'select',
                  },
                },
                description:
                  'Monitor the health of your database to detect issues quickly and improve its performance.',
                label: 'Monitoring',
              },
              resources: {
                components: {
                  nodes: {
                    components: {
                      numberOfnodes: {
                        fieldParams: {
                          defaultValue: 3,
                          label: 'Number of nodes',
                        },
                        path: 'spec.components.engine.replicas',
                        uiType: 'number',
                        validation: {
                          celExpressions: [
                            {
                              celExpr:
                                'spec.components.engine.replicas % 2 == 1',
                              message: 'The number of nodes must be odd',
                            },
                          ],
                          int: true,
                          min: 1,
                          modes: {
                            edit: {
                              celExpressions: [
                                {
                                  celExpr:
                                    '!(spec.components.engine.replicas == 1 && original.spec.components.engine.replicas > 1)',
                                  message: 'Cannot scale down to a single node',
                                },
                              ],
                            },
                          },
                          required: true,
                        },
                      },
                      resources: {
                        components: {
                          cpu: {
                            fieldParams: {
                              defaultValue: 1,
                              label: 'CPU',
                              step: 1,
                            },
                            path: 'spec.components.engine.resources.limits.cpu',
                            uiType: 'number',
                            validation: {
                              min: 0.6,
                              required: true,
                            },
                          },
                          disk: {
                            fieldParams: {
                              badge: 'Gi',
                              badgeToApi: true,
                              defaultValue: 25,
                              label: 'Disk',
                            },
                            path: 'spec.components.engine.storage.size',
                            uiType: 'number',
                            validation: {
                              int: true,
                              min: 1,
                              modes: {
                                edit: {
                                  celExpressions: [
                                    {
                                      celExpr:
                                        'spec.components.engine.storage.size >= original.spec.components.engine.storage.size',
                                      message: 'Disk size cannot be decreased',
                                    },
                                  ],
                                },
                              },
                              required: true,
                            },
                          },
                          memory: {
                            fieldParams: {
                              badge: 'Gi',
                              badgeToApi: true,
                              defaultValue: 4,
                              label: 'Memory',
                              step: 0.1,
                            },
                            path: 'spec.components.engine.resources.limits.memory',
                            uiType: 'number',
                            validation: {
                              min: 0.512,
                              required: true,
                            },
                          },
                        },
                        groupType: 'line',
                        uiType: 'group',
                      },
                    },
                    componentsOrder: ['numberOfnodes', 'resources'],
                    uiType: 'group',
                  },
                },
                description:
                  'Configure the resources your new database will have access to.',
                label: 'Resources',
              },
            },
            sectionsOrder: [
              'databaseVersion',
              'resources',
              'monitoring',
              'advanced',
            ],
          },
          sharded: {
            sections: {
              advanced: {
                components: {
                  configuration: {
                    fieldParams: {
                      label: 'Еngine parameters',
                      maxRows: 8,
                      minRows: 3,
                      multiline: true,
                      placeholder:
                        'operationProfiling:\n  mode: slowOps\n  slowOpThresholdMs: 200\n',
                    },
                    path: 'spec.components.engine.configuration',
                    uiType: 'text',
                  },
                  externalAccess: {
                    fieldParams: {
                      defaultValue: 'ClusterIP',
                      label: 'External access',
                      options: [
                        {
                          label: 'ClusterIP',
                          value: 'ClusterIP',
                        },
                        {
                          label: 'LoadBalancer',
                          value: 'LoadBalancer',
                        },
                        {
                          label: 'NodePort',
                          value: 'NodePort',
                        },
                      ],
                    },
                    path: 'spec.components.proxy.service.serviceType',
                    uiType: 'select',
                  },
                  storageClass: {
                    dataSource: {
                      provider: 'storageClasses',
                    },
                    fieldParams: {
                      label: 'Storage class',
                      modes: {
                        edit: {
                          disabled: true,
                        },
                        restore: {
                          disabled: true,
                        },
                      },
                    },
                    path: 'spec.components.engine.storage.storageClass',
                    uiType: 'select',
                    validation: {
                      required: true,
                    },
                  },
                },
                componentsOrder: ['storageClass', 'configuration'],
                description: 'Configure advanced settings for your database',
                label: 'Advanced configuration',
              },
              databaseVersion: {
                components: {
                  version: {
                    fieldParams: {
                      label: 'Database Version',
                      modes: {
                        edit: {
                          disabled: true,
                        },
                        restore: {
                          disabled: true,
                        },
                      },
                      optionsPath: 'spec.versions',
                      optionsPathConfig: {
                        labelPath: 'name',
                        valuePath: 'name',
                      },
                    },
                    path: 'spec.version',
                    uiType: 'select',
                    validation: {
                      required: true,
                    },
                  },
                },
                description:
                  'Provide the information about the database version you want to use.',
                label: 'Database Version',
              },
              monitoring: {
                components: {
                  monitoring: {
                    dataSource: {
                      provider: 'monitoringConfigs',
                    },
                    fieldParams: {
                      label: 'Monitoring endpoint',
                    },
                    path: 'spec.components.monitoring.customSpec.monitoringConfigName',
                    uiType: 'select',
                  },
                },
                description:
                  'Monitor the health of your database to detect issues quickly and improve its performance.',
                label: 'Monitoring',
              },
              resources: {
                components: {
                  nodesResources: {
                    components: {
                      cpu: {
                        fieldParams: {
                          defaultValue: 1,
                          label: 'CPU',
                          step: 1,
                        },
                        path: 'spec.components.engine.resources.limits.cpu',
                        uiType: 'number',
                        validation: {
                          min: 0.6,
                          required: true,
                        },
                      },
                      disk: {
                        fieldParams: {
                          badge: 'Gi',
                          badgeToApi: true,
                          defaultValue: 25,
                          label: 'Disk',
                        },
                        path: 'spec.components.engine.storage.size',
                        uiType: 'number',
                        validation: {
                          int: true,
                          min: 1,
                          modes: {
                            edit: {
                              celExpressions: [
                                {
                                  celExpr:
                                    'spec.components.engine.storage.size >= original.spec.components.engine.storage.size',
                                  message: 'Disk size cannot be decreased',
                                },
                              ],
                            },
                          },
                          required: true,
                        },
                      },
                      memory: {
                        fieldParams: {
                          badge: 'Gi',
                          badgeToApi: true,
                          defaultValue: 4,
                          label: 'Memory',
                          step: 0.1,
                        },
                        path: 'spec.components.engine.resources.limits.memory',
                        uiType: 'number',
                        validation: {
                          min: 0.512,
                          required: true,
                        },
                      },
                    },
                    groupType: 'line',
                    uiType: 'group',
                  },
                  numberOfConfigServers: {
                    fieldParams: {
                      defaultValue: 3,
                      label: 'Nº of configuration servers',
                    },
                    path: 'spec.components.configServer.replicas',
                    uiType: 'number',
                    validation: {
                      celExpressions: [
                        {
                          celExpr:
                            'spec.components.configServer.replicas % 2 == 1',
                          message:
                            'The number of configuration servers must be odd',
                        },
                        {
                          celExpr:
                            '!(spec.components.engine.replicas > 1 && spec.components.configServer.replicas == 1)',
                          message:
                            'The number of configuration servers cannot be 1 if the number of database nodes is greater than 1',
                        },
                      ],
                      int: true,
                      max: 7,
                      min: 1,
                      required: true,
                    },
                  },
                  numberOfRouters: {
                    fieldParams: {
                      defaultValue: 3,
                      label: 'Number of routers',
                    },
                    path: 'spec.components.proxy.replicas',
                    uiType: 'number',
                    validation: {
                      celExpressions: [
                        {
                          celExpr: 'spec.components.proxy.replicas % 2 == 1',
                          message: 'The number of routers must be odd',
                        },
                        {
                          celExpr:
                            'spec.components.engine.replicas > 1 ? spec.components.proxy.replicas > 1 : true',
                          message:
                            'Number of routers must be more than 1 if number of nodes is greater than 1',
                        },
                      ],
                      int: true,
                      min: 1,
                      required: true,
                    },
                  },
                  numberOfnodes: {
                    fieldParams: {
                      defaultValue: 3,
                      label: 'Number of nodes',
                    },
                    path: 'spec.components.engine.replicas',
                    uiType: 'number',
                    validation: {
                      celExpressions: [
                        {
                          celExpr: 'spec.components.engine.replicas % 2 == 1',
                          message: 'The number of nodes must be odd',
                        },
                      ],
                      int: true,
                      min: 1,
                      modes: {
                        edit: {
                          celExpressions: [
                            {
                              celExpr:
                                '!(spec.components.engine.replicas == 1 && original.spec.components.engine.replicas > 1)',
                              message: 'Cannot scale down to a single node',
                            },
                          ],
                        },
                      },
                      required: true,
                    },
                  },
                  routersResources: {
                    components: {
                      cpu: {
                        fieldParams: {
                          defaultValue: 1,
                          label: 'CPU',
                          step: 1,
                        },
                        path: 'spec.components.proxy.resources.limits.cpu',
                        uiType: 'number',
                        validation: {
                          min: 0,
                          required: true,
                        },
                      },
                      memory: {
                        fieldParams: {
                          badge: 'Gi',
                          badgeToApi: true,
                          defaultValue: 2,
                          label: 'Memory',
                          step: 0.1,
                        },
                        path: 'spec.components.proxy.resources.limits.memory',
                        uiType: 'number',
                        validation: {
                          min: 0,
                          required: true,
                        },
                      },
                    },
                    groupType: 'line',
                    uiType: 'group',
                  },
                  shards: {
                    fieldParams: {
                      defaultValue: 1,
                      label: 'Nº of shards',
                    },
                    path: 'spec.topology.config.numShards',
                    uiType: 'number',
                    validation: {
                      int: true,
                      min: 1,
                      modes: {
                        edit: {
                          celExpressions: [
                            {
                              celExpr:
                                'spec.topology.config.numShards >= original.spec.topology.config.numShards',
                              message: 'Number of shards cannot be decreased',
                            },
                          ],
                        },
                      },
                      required: true,
                    },
                  },
                },
                componentsOrder: [
                  'shards',
                  'numberOfnodes',
                  'nodesResources',
                  'numberOfRouters',
                  'routersResources',
                  'numberOfConfigServers',
                ],
                description:
                  'Configure the resources your new database will have access to.',
                label: 'Resources',
              },
            },
            sectionsOrder: [
              'databaseVersion',
              'resources',
              'monitoring',
              'advanced',
            ],
          },
        },
      },
    },
  ],
};
