import { Instance } from 'types/api';
import { InstanceStatus } from 'shared-types/instance.types';
import { convertDbInstancesPayloadToTableFormat } from './DbClusterView.utils';
import { InstanceTableElement } from './dbClusterView.types';
import { DbInstanceForNamespaceResult } from 'hooks/api/db-instances';

const makeInstance = (
  _name: string,
  provider: string,
  phase: InstanceStatus,
  topologyType: string
): Instance => ({
  apiVersion: 'core.openeverest.io/v1alpha1',
  kind: 'Instance',
  metadata: {},
  spec: {
    provider,
    topology: { type: topologyType },
  },
  status: {
    phase,
  },
});

const makeResult = (
  namespace: string,
  instances: Instance[],
  isSuccess = true
): DbInstanceForNamespaceResult => ({
  namespace,
  queryResult: {
    data: instances,
    isSuccess,
    isLoading: false,
    isFetching: false,
  } as DbInstanceForNamespaceResult['queryResult'],
});

describe('convertDbInstancesPayloadToTableFormat', () => {
  it('returns empty array when no results', () => {
    expect(convertDbInstancesPayloadToTableFormat([])).toEqual([]);
  });

  it('converts a single namespace with one instance', () => {
    const instance = makeInstance('pg-1', 'aws', InstanceStatus.Running, 'ha');
    const data = [makeResult('ns-1', [instance])];

    const result = convertDbInstancesPayloadToTableFormat(data);

    expect(result).toHaveLength(1);
    expect(result[0]).toEqual<InstanceTableElement>({
      namespace: 'ns-1',
      instanceName: '',
      phase: InstanceStatus.Running,
      provider: 'aws',
      topologyType: 'ha',
      raw: instance,
    });
  });

  it('converts multiple namespaces', () => {
    const i1 = makeInstance('pg-1', 'aws', InstanceStatus.Running, 'ha');
    const i2 = makeInstance(
      'mongo-1',
      'gcp',
      InstanceStatus.Creating,
      'standalone'
    );
    const data = [makeResult('ns-1', [i1]), makeResult('ns-2', [i2])];

    const result = convertDbInstancesPayloadToTableFormat(data);

    expect(result).toHaveLength(2);
    expect(result[0].namespace).toBe('ns-1');
    expect(result[1].namespace).toBe('ns-2');
    expect(result[1].provider).toBe('gcp');
  });

  it('skips namespaces where query is not successful', () => {
    const instance = makeInstance('pg-1', 'aws', InstanceStatus.Running, 'ha');
    const data = [makeResult('ns-1', [instance], false)];

    const result = convertDbInstancesPayloadToTableFormat(data);
    expect(result).toHaveLength(0);
  });

  it('defaults phase to creating when status is undefined', () => {
    const instance: Instance = {
      apiVersion: 'core.openeverest.io/v1alpha1',
      kind: 'Instance',
      metadata: {},
      spec: { provider: 'aws' },
    };
    const data = [makeResult('ns-1', [instance])];

    const result = convertDbInstancesPayloadToTableFormat(data);
    expect(result[0].phase).toBe(InstanceStatus.Creating);
  });

  it('defaults provider to empty string when not set', () => {
    const instance: Instance = {
      apiVersion: 'core.openeverest.io/v1alpha1',
      kind: 'Instance',
      metadata: {},
      spec: {},
    };
    const data = [makeResult('ns-1', [instance])];

    const result = convertDbInstancesPayloadToTableFormat(data);
    expect(result[0].provider).toBe('');
  });
});
