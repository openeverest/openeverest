import { TopologyUISchemas, GroupType, FieldType } from './ui-generator.types';
// Note: This file is using for development and testing purposes, it contains mock data for the UI generator component.
// TODO It should be removed in production.
export const topologyUiSchemas: TopologyUISchemas = {
  replica: {
    sections: {
      basicInfo: {
        label: 'Basic Information',
        description: 'Provide the basic information for your new database.',
        components: {
          version: {
            uiType: FieldType.Select as const,
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
        label: 'Resources',
        description:
          'Configure the resources your new database will have access to.',
        components: {
          numberOfnodes: {
            path: 'spec.replica.nodes',
            uiType: FieldType.Number as const, // RadioButtons/Number/even Select
            fieldParams: {
              label: 'Number of nodes',
            },
            validation: {
              min: 1,
              max: 7,
            },
          },
          resources: {
            uiType: 'group' as const,
            groupType: GroupType.Line as const,
            components: {
              cpu: {
                path: 'spec.engine.resources.cpu',
                uiType: FieldType.Number as const,
                fieldParams: {
                  label: 'CPU',
                },
                validation: {
                  min: 1,
                  max: 10,
                },
              },
              memory: {
                path: 'spec.engine.resources.memory',
                uiType: FieldType.Number as const,
                fieldParams: {
                  label: 'Memory',
                },
                validation: {
                  min: 1,
                  max: 10,
                },
              },
              disk: {
                path: 'spec.engine.resources.disk',
                uiType: FieldType.Number as const,
                fieldParams: {
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
} satisfies TopologyUISchemas;
