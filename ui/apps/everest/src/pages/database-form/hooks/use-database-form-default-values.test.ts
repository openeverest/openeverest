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
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { buildImportNavigationRouteState } from '../import-navigation-state';
import { useDatabasePageDefaultValues } from './use-database-form-default-values';

const ROUTER_STATE = buildImportNavigationRouteState(
  { metadata: { name: 'psmdb' }, spec: {} },
  {
    namespace: 'ns-b',
    backupClassName: 'psmdb-pbm',
    dataSource: {
      type: 'Backup',
      backupName: 'import-abc12-1',
      storageName: 's3-new',
    },
  }
);

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useLocation: () => ({ state: ROUTER_STATE }),
}));
vi.mock('hooks/api/db-instances/useDbInstance', () => ({
  useDbInstance: () => ({ status: 'pending', isFetching: false }),
}));

describe('useDatabasePageDefaultValues', () => {
  it('prefills an import with the namespace and class of the backup', () => {
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
});
