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
} from 'components/ui-generator/ui-generator.types';
import { buildRestoreNavigationRouteState } from '../restore-navigation-state';
import { useSchema } from './use-schema';

const mocks = vi.hoisted(() => ({
  state: undefined as unknown,
  source: { spec: { providerRef: { name: 'psmdb' } } },
  providers: [
    {
      metadata: { name: 'psmdb' },
      spec: {
        uiSchema: {
          single: {
            sections: {
              base: {
                components: {
                  users: {
                    uiType: 'text',
                    path: 'spec.userSecretRef.name',
                    fieldParams: { label: 'Users' },
                    modes: { restore: { uiType: 'hidden' } },
                  },
                },
              },
            },
          },
        },
      },
    },
  ],
}));

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useLocation: () => ({ state: mocks.state }),
}));
vi.mock('hooks/api/db-instances/useDbInstance', () => ({
  useDbInstance: () => ({ data: mocks.source }),
}));
vi.mock('hooks/api/providers/useProviders', () => ({
  useProviders: () => ({ data: mocks.providers }),
}));

describe('useSchema', () => {
  it.each([
    [FormMode.Restore, 'hidden'],
    [FormMode.New, FieldType.Text],
  ])('resolves the schema for %s', (mode, uiType) => {
    mocks.state =
      mode === FormMode.Restore
        ? buildRestoreNavigationRouteState('source-db', 'ns-a', {
            type: 'Backup',
            backupName: 'daily-1',
          })
        : { selectedDbProvider: mocks.providers[0] };

    const { result } = renderHook(() => useSchema());

    expect(
      result.current.uiSchema.single.sections.base.components.users.uiType
    ).toBe(uiType);
  });
});
