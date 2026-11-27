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

import { renderHook } from '@testing-library/react';
import {
  FieldType,
  FormMode,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { keyValueRow } from 'components/ui-generator/test-fixtures/key-value-row.fixture';
import { Provider } from 'shared-types/api.types';
import {
  buildImportNavigationRouteState,
  ImportSource,
} from '../import-navigation-state';
import { buildRestoreNavigationRouteState } from '../restore-navigation-state';
import { useDatabasePageDefaultValues } from './use-database-form-default-values';

const PROVIDER: Provider = {
  metadata: { name: 'psmdb' },
  spec: {
    versions: [{ name: '7.0.18', components: { engine: '7.0.18-11' } }],
  },
};
const IMPORT_SOURCE: ImportSource = {
  namespace: 'ns-b',
  backupClassName: 'psmdb-pbm',
  dataSource: {
    type: 'Backup',
    backupName: 'import-abc12-1',
    storageName: 's3-new',
  },
};
const UI_SCHEMA: TopologyUISchemas = {
  replicaset: { sections: {} },
  sharded: { sections: {} },
};

const mocks = vi.hoisted((): { state: unknown; sourceInstance: unknown } => ({
  state: undefined,
  sourceInstance: undefined,
}));

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useLocation: () => ({ state: mocks.state }),
}));
vi.mock('hooks/api/db-instances/useDbInstance', () => ({
  useDbInstance: () => ({
    data: mocks.sourceInstance,
    status: 'pending',
    isFetching: false,
  }),
}));

describe('useDatabasePageDefaultValues', () => {
  it("inherits the source's monitoring destinations, parameters included, on restore", () => {
    const destinations = [
      {
        name: 'pmm',
        destinationRef: { name: 'pmm-prod' },
        parameters: { resources: { requests: { cpu: '100m' } } },
      },
    ];
    mocks.state = buildRestoreNavigationRouteState('orders', 'team-a', {
      type: 'Backup',
      backupName: 'orders-backup',
    });
    mocks.sourceInstance = {
      metadata: { name: 'orders', namespace: 'team-a' },
      spec: {
        providerRef: { name: 'psmdb' },
        topology: { type: 'replicaset' },
        monitoring: { destinations },
      },
    };
    const { result } = renderHook(() =>
      useDatabasePageDefaultValues(FormMode.Restore, UI_SCHEMA, 'replicaset')
    );
    mocks.sourceInstance = undefined;

    expect(result.current.defaultValues).toMatchObject({
      k8sNamespace: 'team-a',
      monitoring: { destinations },
    });
  });

  describe('with a map and a list that have nothing to show', () => {
    const schema: TopologyUISchemas = {
      replicaset: {
        sections: {
          advanced: {
            components: {
              labels: {
                uiType: FieldType.Map,
                path: 'spec.labels',
                fieldParams: { label: 'Labels' },
                itemTemplate: keyValueRow(),
              },
              args: {
                uiType: FieldType.List,
                path: 'spec.args',
                fieldParams: { label: 'Arguments' },
                itemTemplate: { uiType: FieldType.Text },
              },
            },
          },
        },
      },
    };
    // The form starts from these values, so it is unchanged while they stand.
    const ONE_EMPTY_ROW = {
      spec: {
        labels: [expect.objectContaining({ key: '', value: '' })],
        args: [expect.objectContaining({ value: '' })],
      },
    };

    it.each([FormMode.New, FormMode.Import])(
      'opens the %s form with one empty row in each',
      (mode) => {
        mocks.state =
          mode === FormMode.Import
            ? buildImportNavigationRouteState(PROVIDER, IMPORT_SOURCE)
            : undefined;
        const { result } = renderHook(() =>
          useDatabasePageDefaultValues(mode, schema, 'replicaset')
        );

        expect(result.current.defaultValues).toMatchObject(ONE_EMPTY_ROW);
      }
    );

    it('opens a restore from an instance without them the same way', () => {
      mocks.state = buildRestoreNavigationRouteState('orders', 'team-a', {
        type: 'Backup',
        backupName: 'orders-backup',
      });
      mocks.sourceInstance = {
        metadata: { name: 'orders', namespace: 'team-a' },
        spec: {
          providerRef: { name: 'psmdb' },
          topology: { type: 'replicaset' },
        },
      };
      const { result } = renderHook(() =>
        useDatabasePageDefaultValues(FormMode.Restore, schema, 'replicaset')
      );
      mocks.sourceInstance = undefined;

      expect(result.current.defaultValues).toMatchObject(ONE_EMPTY_ROW);
    });
  });

  it('prefills an import with the namespace and class of the backup', () => {
    mocks.state = buildImportNavigationRouteState(PROVIDER, IMPORT_SOURCE);
    const { result } = renderHook(() =>
      useDatabasePageDefaultValues(FormMode.Import, {}, 'replicaset')
    );

    expect(result.current.defaultValues).toMatchObject({
      provider: 'psmdb',
      k8sNamespace: 'ns-b',
      topology: { type: 'replicaset' },
      backup: { schedules: [], classRef: { name: 'psmdb-pbm' } },
    });
  });

  it('prefills the topology and version the backup was taken on', () => {
    mocks.state = buildImportNavigationRouteState(PROVIDER, {
      ...IMPORT_SOURCE,
      sourceMetadata: { engineVersion: '7.0.18-11', topology: 'sharded' },
    });
    const { result } = renderHook(() =>
      useDatabasePageDefaultValues(
        FormMode.Import,
        UI_SCHEMA,
        'replicaset',
        false,
        PROVIDER
      )
    );

    expect(result.current.defaultValues).toMatchObject({
      topology: { type: 'sharded' },
      spec: { version: '7.0.18' },
    });
  });
});
