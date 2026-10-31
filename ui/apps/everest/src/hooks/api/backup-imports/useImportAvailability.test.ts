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
import { BackupClass } from 'shared-types/backups.types';
import { RBACResource } from 'utils/rbac';
import { useImportAvailability } from './useImportAvailability';

interface NamespacePermissions {
  canCreate: string[];
  canRead: string[];
  isLoading: boolean;
  isPermissionsLoading: boolean;
}

const IMPORT_CLASS: BackupClass = {
  metadata: { name: 'psmdb-pbm' },
  spec: {
    executionMode: 'ProviderManaged',
    supportsImport: true,
    backupStorageClasses: [],
  },
};

const mocks = vi.hoisted(() => {
  const state: {
    permissions: Partial<Record<string, unknown>>;
    classes: { data: unknown[]; isLoading: boolean };
  } = { permissions: {}, classes: { data: [], isLoading: false } };
  return state;
});

vi.mock('hooks/api/useClusterName', () => ({ useClusterName: () => 'main' }));
vi.mock('hooks/api/backup-classes/useBackupClasses', () => ({
  useImportBackupClasses: () => mocks.classes,
}));
vi.mock('hooks/rbac', () => ({
  useNamespacePermissionsForResource: (resource: RBACResource) =>
    mocks.permissions[resource],
}));

const resolved = (
  canCreate: string[],
  canRead: string[] = []
): NamespacePermissions => ({
  canCreate,
  canRead,
  isLoading: false,
  isPermissionsLoading: false,
});

const setUp = ({
  imports,
  instances,
  backups = resolved([], ['ns-a', 'ns-b']),
  restores = resolved(['ns-a', 'ns-b']),
  classes = { data: [IMPORT_CLASS], isLoading: false },
}: {
  imports: NamespacePermissions;
  instances: NamespacePermissions;
  backups?: NamespacePermissions;
  restores?: NamespacePermissions;
  classes?: { data: BackupClass[]; isLoading: boolean };
}) => {
  mocks.permissions = {
    'backup-imports': imports,
    instances,
    backups,
    restores,
  };
  mocks.classes = classes;
  return renderHook(useImportAvailability).result.current;
};

describe('useImportAvailability', () => {
  it('allows import only where the user can scan, follow the scan and create the instance', () => {
    const forbidden = setUp({
      imports: resolved(['ns-a', 'ns-b'], ['ns-a']),
      instances: resolved(['ns-b']),
    });
    const allowed = setUp({
      imports: resolved(['ns-a', 'ns-b'], ['ns-a', 'ns-b']),
      instances: resolved(['ns-b']),
    });

    expect(forbidden).toEqual({ canImport: false, isLoading: false });
    expect(allowed).toEqual({ canImport: true, isLoading: false });
  });

  it('needs to read the found backups and create a restore in the namespace', () => {
    const scanAndCreate = {
      imports: resolved(['ns-a'], ['ns-a']),
      instances: resolved(['ns-a']),
    };

    expect(
      setUp({ ...scanAndCreate, backups: resolved([], ['ns-b']) })
    ).toEqual({ canImport: false, isLoading: false });
    expect(setUp({ ...scanAndCreate, restores: resolved(['ns-b']) })).toEqual({
      canImport: false,
      isLoading: false,
    });
  });

  it('needs a backup class that supports import', () => {
    expect(
      setUp({
        imports: resolved(['ns-a'], ['ns-a']),
        instances: resolved(['ns-a']),
        classes: { data: [], isLoading: false },
      })
    ).toEqual({ canImport: false, isLoading: false });
  });

  it('is loading until permissions and classes are known', () => {
    expect(
      setUp({
        imports: { ...resolved([]), isPermissionsLoading: true },
        instances: resolved([]),
      }).isLoading
    ).toBe(true);
    expect(
      setUp({
        imports: resolved(['ns-a'], ['ns-a']),
        instances: resolved(['ns-a']),
        classes: { data: [], isLoading: true },
      }).isLoading
    ).toBe(true);
  });

  it("doesn't wait for classes when there is no namespace to import into", () => {
    expect(
      setUp({
        imports: resolved([]),
        instances: resolved(['ns-a']),
        classes: { data: [], isLoading: true },
      })
    ).toEqual({ canImport: false, isLoading: false });
  });
});
