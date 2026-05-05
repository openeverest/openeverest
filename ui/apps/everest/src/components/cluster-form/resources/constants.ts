import { DbType } from '@percona/types';
import { z } from 'zod';
import { Resources } from 'shared-types/dbCluster.types';
import { DbWizardFormFields } from 'consts';
import { cpuParser, memoryParser } from 'utils/k8ResourceParser';
import { Messages } from './messages';
import { isVersion84x } from './utils';

const resourceToNumber = (minimum = 0) =>
  z.union([z.string().min(1), z.number()]).pipe(
    z.coerce
      .number({
        invalid_type_error: 'Please enter a valid number',
      })
      .min(minimum)
  );

export const matchFieldsValueToResourceSize = (
  sizes: Record<
    Exclude<ResourceSize, ResourceSize.custom>,
    Record<'cpu' | 'memory', number>
  >,
  resources?: Resources
): ResourceSize => {
  if (!resources) {
    return ResourceSize.custom;
  }
  const memory = memoryParser(resources.memory.toString(), 'G');
  const res = Object.values(sizes).findIndex((item) => {
    const sizeParsedMemory = memoryParser(item.memory.toString(), 'G');
    return (
      cpuParser(item.cpu.toString()) === cpuParser(resources.cpu.toString()) &&
      sizeParsedMemory.value === memory.value
    );
  });
  return res !== -1
    ? (Object.keys(sizes)[res] as ResourceSize)
    : ResourceSize.custom;
};

export const NODES_DB_TYPE_MAP: Record<DbType, string[]> = {
  [DbType.Mongo]: ['1', '3', '5'],
  [DbType.Mysql]: ['1', '3', '5'],
  [DbType.Postresql]: ['1', '2', '3'],
};

export enum ResourceSize {
  small = 'small',
  medium = 'medium',
  large = 'large',
  custom = 'custom',
}

export const humanizedResourceSizeMap: Record<ResourceSize, string> = {
  [ResourceSize.small]: 'Small',
  [ResourceSize.medium]: 'Medium',
  [ResourceSize.large]: 'Large',
  [ResourceSize.custom]: 'Custom',
};

export const NODES_DEFAULT_SIZES = (dbType: DbType, dbVersion: string = '') => {
  switch (dbType) {
    case DbType.Mysql:
      return {
        [ResourceSize.small]: {
          [DbWizardFormFields.cpu]: 1,
          [DbWizardFormFields.memory]: isVersion84x(dbVersion) ? 3 : 2,
          [DbWizardFormFields.disk]: 25,
        },
        [ResourceSize.medium]: {
          [DbWizardFormFields.cpu]: 4,
          [DbWizardFormFields.memory]: 8,
          [DbWizardFormFields.disk]: 100,
        },
        [ResourceSize.large]: {
          [DbWizardFormFields.cpu]: 8,
          [DbWizardFormFields.memory]: 32,
          [DbWizardFormFields.disk]: 200,
        },
      };
    case DbType.Mongo:
      return {
        [ResourceSize.small]: {
          [DbWizardFormFields.cpu]: 1,
          [DbWizardFormFields.memory]: 4,
          [DbWizardFormFields.disk]: 25,
        },
        [ResourceSize.medium]: {
          [DbWizardFormFields.cpu]: 4,
          [DbWizardFormFields.memory]: 8,
          [DbWizardFormFields.disk]: 100,
        },
        [ResourceSize.large]: {
          [DbWizardFormFields.cpu]: 8,
          [DbWizardFormFields.memory]: 32,
          [DbWizardFormFields.disk]: 200,
        },
      };
    case DbType.Postresql:
      return {
        [ResourceSize.small]: {
          [DbWizardFormFields.cpu]: 1,
          [DbWizardFormFields.memory]: 2,
          [DbWizardFormFields.disk]: 25,
        },
        [ResourceSize.medium]: {
          [DbWizardFormFields.cpu]: 4,
          [DbWizardFormFields.memory]: 8,
          [DbWizardFormFields.disk]: 100,
        },
        [ResourceSize.large]: {
          [DbWizardFormFields.cpu]: 8,
          [DbWizardFormFields.memory]: 32,
          [DbWizardFormFields.disk]: 200,
        },
      };
  }
};

export const PROXIES_DEFAULT_SIZES = {
  [DbType.Mysql]: {
    [ResourceSize.small]: {
      [DbWizardFormFields.cpu]: 0.2,
      [DbWizardFormFields.memory]: 0.2,
    },
    [ResourceSize.medium]: {
      [DbWizardFormFields.cpu]: 0.5,
      [DbWizardFormFields.memory]: 0.8,
    },
    [ResourceSize.large]: {
      [DbWizardFormFields.cpu]: 0.8,
      [DbWizardFormFields.memory]: 3,
    },
  },
  [DbType.Mongo]: {
    [ResourceSize.small]: {
      [DbWizardFormFields.cpu]: 1,
      [DbWizardFormFields.memory]: 2,
    },
    [ResourceSize.medium]: {
      [DbWizardFormFields.cpu]: 2,
      [DbWizardFormFields.memory]: 4,
    },
    [ResourceSize.large]: {
      [DbWizardFormFields.cpu]: 4,
      [DbWizardFormFields.memory]: 16,
    },
  },
  [DbType.Postresql]: {
    [ResourceSize.small]: {
      [DbWizardFormFields.cpu]: 1,
      [DbWizardFormFields.memory]: 0.03,
    },
    [ResourceSize.medium]: {
      [DbWizardFormFields.cpu]: 4,
      [DbWizardFormFields.memory]: 0.06,
    },
    [ResourceSize.large]: {
      [DbWizardFormFields.cpu]: 8,
      [DbWizardFormFields.memory]: 0.1,
    },
  },
};

export const DEFAULT_CONFIG_SERVERS = [1, 3, 5, 7];

export const MIN_NUMBER_OF_SHARDS = '1';

export const getDefaultNumberOfconfigServersByNumberOfNodes = (
  numberOfNodes: number
) => {
  if (DEFAULT_CONFIG_SERVERS.includes(numberOfNodes)) {
    return numberOfNodes;
  } else {
    return 7;
  }
};

const numberOfResourcesValidator = (
  numberOfResourcesStr: string,
  customNrOfResoucesStr: string,
  fieldPath: string,
  ctx: z.RefinementCtx
) => {
  if (numberOfResourcesStr === CUSTOM_NR_UNITS_INPUT_VALUE) {
    const intNr = parseInt(customNrOfResoucesStr, 10);

    if (Number.isNaN(intNr) || intNr < 1) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Please enter a valid number',
        path: [fieldPath],
      });
    }
  }
};

export const resourcesFormSchema = (
  defaultValues: Record<string, unknown>,
  allowShardingDescaling: boolean,
  allowDescalingToOneNode: boolean,
  allowDiskDescaling: boolean
) => {
  const objectShape = {
    [DbWizardFormFields.shardNr]: z.string().optional(),
    [DbWizardFormFields.shardConfigServers]: z.number().optional(),
    [DbWizardFormFields.cpu]: resourceToNumber(0.6),
    [DbWizardFormFields.memory]: resourceToNumber(0.512),
    [DbWizardFormFields.disk]: resourceToNumber(1),
    // we will never input this, but we need it and zod will let it pass
    [DbWizardFormFields.diskUnit]: z.string(),
    [DbWizardFormFields.resourceSizePerNode]: z.nativeEnum(ResourceSize),
    [DbWizardFormFields.numberOfNodes]: z.string(),
    [DbWizardFormFields.customNrOfNodes]: z.string().optional(),
    [DbWizardFormFields.proxyCpu]: resourceToNumber(0),
    [DbWizardFormFields.proxyMemory]: resourceToNumber(0),
    [DbWizardFormFields.resourceSizePerProxy]: z.nativeEnum(ResourceSize),
    [DbWizardFormFields.numberOfProxies]: z.string(),
    [DbWizardFormFields.customNrOfProxies]: z.string().optional(),
  };

  const zObject = z.object(objectShape).passthrough();

  return zObject.superRefine(
    (
      {
        sharding,
        shardConfigServers,
        shardNr,
        numberOfNodes,
        numberOfProxies,
        customNrOfNodes = '',
        customNrOfProxies = '',
        dbType,
        disk,
      },
      ctx
    ) => {
      const areNodesCustom = numberOfNodes === CUSTOM_NR_UNITS_INPUT_VALUE;
      const areProxiesCustom = numberOfProxies === CUSTOM_NR_UNITS_INPUT_VALUE;
      const customNrOfNodesInt = parseInt(customNrOfNodes, 10);

      numberOfResourcesValidator(
        numberOfNodes,
        customNrOfNodes,
        DbWizardFormFields.customNrOfNodes,
        ctx
      );

      if (dbType !== DbType.Mongo || (dbType === DbType.Mongo && !!sharding)) {
        numberOfResourcesValidator(
          numberOfProxies,
          customNrOfProxies,
          DbWizardFormFields.customNrOfProxies,
          ctx
        );
      }

      if (
        areNodesCustom &&
        dbType === DbType.Mongo &&
        customNrOfNodesInt % 2 === 0
      ) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'The number of nodes cannot be even',
          path: [DbWizardFormFields.customNrOfNodes],
        });
      }

      const intNrNodes = areNodesCustom
        ? customNrOfNodesInt
        : parseInt(numberOfNodes, 10);

      if (dbType === DbType.Mysql) {
        const intNrProxies = parseInt(
          areProxiesCustom ? customNrOfProxies : numberOfProxies,
          10
        );

        if (intNrNodes > 1 && intNrProxies === 1) {
          if (areProxiesCustom) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: 'Number of proxies must be more than 1',
              path: [DbWizardFormFields.customNrOfProxies],
            });
          } else {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: 'Number of proxies must be more than 1',
              path: [DbWizardFormFields.numberOfProxies],
            });
          }
        }
      }

      if (!allowDescalingToOneNode) {
        const prevNumberOfNodes = defaultValues[
          DbWizardFormFields.numberOfNodes
        ] as string;

        const prevNumberOfNodesInt =
          prevNumberOfNodes === CUSTOM_NR_UNITS_INPUT_VALUE
            ? parseInt(
                defaultValues[DbWizardFormFields.customNrOfNodes] as string,
                10
              )
            : parseInt(prevNumberOfNodes, 10);

        if (intNrNodes === 1 && prevNumberOfNodesInt > 1) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: 'Cannot scale down to one node.',
            path: [DbWizardFormFields.numberOfNodes],
          });
        }
      }

      if (sharding as boolean) {
        const intShardNr = parseInt(shardNr || '', 10);
        const intShardNrMin = +MIN_NUMBER_OF_SHARDS;
        const intShardConfigServers = shardConfigServers;

        if (Number.isNaN(intShardNr) || intShardNr < 0) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: Messages.sharding.invalid,
            path: [DbWizardFormFields.shardNr],
          });
        } else {
          const previousSharding = defaultValues[
            DbWizardFormFields.shardNr
          ] as string;
          const intPreviousSharding = parseInt(previousSharding || '', 10);

          if (intShardNr < intShardNrMin) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: Messages.sharding.min(intShardNrMin),
              path: [DbWizardFormFields.shardNr],
            });
          }

          // TODO test the following:
          // If sharding is enabled, the number of shards cannot be decreased via edit
          if (!allowShardingDescaling && intShardNr < intPreviousSharding) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: [DbWizardFormFields.shardNr],
              message: Messages.descaling,
            });
          }
        }

        if (
          !Number.isNaN(numberOfNodes) &&
          numberOfNodes !== CUSTOM_NR_UNITS_INPUT_VALUE
        ) {
          if (intShardConfigServers === 1 && +numberOfNodes > 1) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: Messages.sharding.numberOfConfigServersError,
              path: [DbWizardFormFields.shardConfigServers],
            });
          }
        } else {
          if (!Number.isNaN(customNrOfNodes)) {
            if (intShardConfigServers === 1 && +customNrOfNodes > 1) {
              ctx.addIssue({
                code: z.ZodIssueCode.custom,
                message: Messages.sharding.numberOfConfigServersError,
                path: [DbWizardFormFields.shardConfigServers],
              });
            }
          }
        }
      }

      const prevDiskValue = defaultValues[DbWizardFormFields.disk] as number;
      if (!allowDiskDescaling && disk < prevDiskValue) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: Messages.descaling,
          path: [DbWizardFormFields.disk],
        });
      }

      if (!Number.isInteger(disk)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: Messages.integerNumber,

          path: [DbWizardFormFields.disk],
        });
      }
    }
  );
};

export const CUSTOM_NR_UNITS_INPUT_VALUE = 'custom-units-nr';
