import { TopologyUISchemas, GroupType } from './ui-generator.types';

export const topologyUiSchemas: TopologyUISchemas = {
  replica: {
    sections: {
      basicInfo: {
        name: 'Basic Information',
        description: 'Provide the basic information for your new database.',
        components: {
          version: {
            uiType: 'select', //it can be autocompleteselect?
            path: 'spec.engine.version',
            fieldParams: {
              label: 'Database Version',
              // TODO CHECK WITH THE TEAM: in case of dbVersions we are assume that we get availableVersions values already
              // or we need to think about an extra logic an it will be special component like:
              // VersionSelect or DbVersionSelect
              options: [
                {
                  label: 'percona/percona-server-mongodb:6.0.19-16-multi',
                  value: '6.0.19-16',
                },
                {
                  label: 'percona/percona-server-mongodb:6.0.21-18',
                  value: '6.0.21-18',
                },
                {
                  label: 'percona/percona-server-mongodb:7.0.18-11',
                  value: '7.0.18-11',
                },
              ],
            },
          },
        },
      },
      resources: {
        name: 'Resources',
        description:
          'Configure the resources your new database will have access to.',
        components: {
          numberOfnodes: {
            path: 'spec.replica.nodes',
            uiType: 'number', // RadioButtons/Number/even Select
            fieldParams: {
              label: 'Number of nodes',
            },
            validation: {
              min: 1,
              max: 7,
            },
          },
          resources: {
            uiType: 'group',
            groupType: GroupType.Line,
            components: {
              cpu: {
                path: 'spec.engine.resources.cpu',
                uiType: 'number',
                fieldParams: {
                  badge: 'CPU',
                  label: 'CPU',
                },
                validation: {
                  min: 1,
                  max: 10,
                },
              },
              memory: {
                path: 'spec.engine.resources.memory',
                uiType: 'number',
                fieldParams: {
                  badge: 'Gi',
                  label: 'Memory',
                },
                validation: {
                  min: 1,
                  max: 10,
                },
              },
              disk: {
                path: 'spec.engine.resources.memory',
                uiType: 'number',
                fieldParams: {
                  badge: 'Gi',
                  label: 'Disk',
                },
                validation: {
                  min: 10,
                  max: 100,
                },
              },
            },
          },
        },
      },
    },
    //backups
    //advanced
    sectionsOrder: ['basicInfo', 'resources'],
  },
  sharded: {
    sections: {
      basicInfo: {
        name: 'Basic Information',
        description: 'Provide the basic information for your new database.',
        components: {
          version: {
            uiType: 'select', //it can be autocompleteselect?
            path: 'spec.engine.version',
            fieldParams: {
              label: 'Database Version',
              options: [
                {
                  label: 'percona/percona-server-mongodb:6.0.19-16-multi',
                  value: '6.0.19-16',
                },
                {
                  label: 'percona/percona-server-mongodb:6.0.21-18',
                  value: '6.0.21-18',
                },
                {
                  label: 'percona/percona-server-mongodb:7.0.18-11',
                  value: '7.0.18-11',
                },
              ],
            },
          },
        },
      },
      resources: {
        name: 'Resources',
        description: 'Some description about resources',
        components: {
          shards: {
            uiType: 'number',
            path: 'spec.sharding.shards',
            fieldParams: {
              label: 'Nº of shards',
              defaultValue: 1,
            },
          },
          numberOfnodes: {
            path: 'spec.replica.nodes',
            uiType: 'number', // RadioButtons/Number/even Select
            fieldParams: {
              label: 'Number of nodes',
            },
          },
          numberOfConfigServers: {
            uiType: 'number', //can be something like NumberTabs or custom type
            path: 'spec.sharding.configServer.replicas',
            fieldParams: {
              label: 'Nº of configuration servers',
              defaultValue: 3,
            },
            validation: {
              celExpressions: [
                {
                  celExpr:
                    '!(spec.replica.nodes > 1 && spec.sharding.configServer.replicas == 1)',
                  message:
                    'The number of configuration servers cannot be 1 if the number of database nodes is greater than 1',
                },
              ],
            },
          },
          },
        },
        componentsOrder: ['shards', 'numberOfnodes', 'numberOfConfigServers'],
      },
      resources2: {
        name: 'Resources',
        description: 'Some description about resources',
        components: {
          shards: {
            uiType: 'number',
            path: 'spec.sharding.shards2',
            fieldParams: {
              label: 'Nº of shards',
              defaultValue: 1,
            },
          },
          numberOfnodes: {
            path: 'spec.replica.nodes2',
            uiType: 'number', // RadioButtons/Number/even Select
            fieldParams: {
              label: 'Number of nodes',
              defaultValue: 3,
            },
          },
          numberOfConfigServers: {
            uiType: 'number', //can be something like NumberTabs or custom type
            path: 'spec.sharding.configServer.replicas2',
            fieldParams: {
              label: 'Nº of configuration servers',
              defaultValue: 3,
            },
          },
        },
        componentsOrder: ['numberOfnodes', 'numberOfConfigServers', 'shards'],
      },
    },
  },
  shardedInOneStep: {
    sections: {
      databaseInfo: {
        name: 'Basic Information',
        description: 'Fill the information for your new database.',
        components: {
          version: {
            uiType: 'select', //it can be autocompleteselect?
            path: 'spec.engine.version',
            fieldParams: {
              label: 'Database Version',
              options: [
                {
                  label: 'percona/percona-server-mongodb:6.0.19-16-multi',
                  value: '6.0.19-16',
                },
                {
                  label: 'percona/percona-server-mongodb:6.0.21-18',
                  value: '6.0.21-18',
                },
                {
                  label: 'percona/percona-server-mongodb:7.0.18-11',
                  value: '7.0.18-11',
                },
              ],
            },
          },
          resources: {
            uiType: 'group',
            groupType: GroupType.Accordion,
            name: 'Resources', //TODO could be could be not
            // description: 'Some description about resources',
            components: {
              shards: {
                uiType: 'number',
                path: 'spec.sharding.shards',
                fieldParams: {
                  label: 'Nº of shards',
                  defaultValue: 1,
                },
              },
              numberOfnodes: {
                path: 'spec.replica.nodes',
                uiType: 'number', // RadioButtons/Number/even Select
                fieldParams: {
                  label: 'Number of nodes',
                },
              },
              numberOfConfigServers: {
                uiType: 'number', //can be something like NumberTabs or custom type
                path: 'spec.sharding.configServer.replicas',
                fieldParams: {
                  label: 'Nº of configuration servers',
                  defaultValue: 3,
                },
              },
            },
            componentsOrder: [
              'shards',
              'numberOfnodes',
              'numberOfConfigServers',
            ],
          },
        },
      },
    },
  },
};
