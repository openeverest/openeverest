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

import { describe, it, expect, vi } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useBackupClassesOptions } from './useBackupClassesOptions';

const useBackupClassesList = vi.fn();
const classes = {
  data: [{ metadata: { name: 'milvus-backup' } }, { metadata: {} }],
  isLoading: false,
  error: null,
};

vi.mock('./useBackupClasses', () => ({
  useBackupClassesList: (cluster: string) => useBackupClassesList(cluster),
}));

describe('useBackupClassesOptions', () => {
  it('offers the cluster-wide BackupClasses whatever the form namespace', () => {
    useBackupClassesList.mockReturnValue(classes);

    const { result } = renderHook(() =>
      useBackupClassesOptions({ namespace: 'vectors', cluster: 'main' })
    );

    expect(useBackupClassesList).toHaveBeenCalledWith('main');
    expect(result.current.options).toEqual([
      { label: 'milvus-backup', value: 'milvus-backup' },
    ]);
  });
});
