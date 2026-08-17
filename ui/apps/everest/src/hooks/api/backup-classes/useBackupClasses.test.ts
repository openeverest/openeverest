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
import { useBackupClassUiSchema } from './useBackupClasses';

const classWithUiSchema = (uiSchema: unknown): BackupClass =>
  ({ spec: { uiSchema } }) as unknown as BackupClass;

const pitrSection = {
  label: 'PITR Configuration',
  componentsOrder: ['timeBetweenUploads'],
  components: { timeBetweenUploads: { uiType: 'number', path: 'x' } },
};
const backupSection = {
  label: 'Backup Configuration',
  componentsOrder: [],
  components: {},
};

describe('useBackupClassUiSchema', () => {
  it('exposes both backup (as parameters) and pitr sections when present', () => {
    const { result } = renderHook(() =>
      useBackupClassUiSchema(
        classWithUiSchema({ backup: backupSection, pitr: pitrSection })
      )
    );

    expect(result.current.sections?.parameters).toEqual(backupSection);
    expect(result.current.sections?.pitr).toEqual(pitrSection);
  });

  it('exposes pitr independently of backup', () => {
    const { result } = renderHook(() =>
      useBackupClassUiSchema(classWithUiSchema({ pitr: pitrSection }))
    );

    expect(result.current.sections?.pitr).toEqual(pitrSection);
    expect(result.current.sections?.parameters).toBeUndefined();
  });

  it('returns no sections when the class ships no uiSchema (on/off provider)', () => {
    const { result } = renderHook(() =>
      useBackupClassUiSchema(classWithUiSchema(undefined))
    );

    expect(result.current.sections).toBeUndefined();
  });
});
