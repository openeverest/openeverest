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

import { FlattenedSchedule } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog-context.types';
import { WizardBackupSpec } from './backup-step.types';
import { buildBackupSpecFromWizard } from './backup-step.utils';

const schedule = (
  storageName: string,
  overrides: Partial<FlattenedSchedule> = {}
): FlattenedSchedule => ({
  name: `sch-${storageName}`,
  cron: '0 2 * * *',
  enabled: true,
  storageName,
  ...overrides,
});

const storagesOf = (spec: WizardBackupSpec | undefined) => spec?.storages ?? [];

describe('buildBackupSpecFromWizard', () => {
  it('returns undefined without schedules or a class', () => {
    expect(buildBackupSpecFromWizard([], 'cls')).toBeUndefined();
    expect(
      buildBackupSpecFromWizard([schedule('s3')], undefined)
    ).toBeUndefined();
  });

  it('attaches enabled PITR to the matching storage', () => {
    const spec = buildBackupSpecFromWizard([schedule('s3')], 'cls', {
      s3: { enabled: true },
    });

    expect(storagesOf(spec)[0].pitr).toEqual({ enabled: true });
  });

  it('includes parameters when the entry has them', () => {
    const spec = buildBackupSpecFromWizard([schedule('s3')], 'cls', {
      s3: { enabled: true, parameters: { timeBetweenUploads: 60 } },
    });

    expect(storagesOf(spec)[0].pitr).toEqual({
      enabled: true,
      parameters: { timeBetweenUploads: 60 },
    });
  });

  it('omits pitr for a disabled entry', () => {
    const spec = buildBackupSpecFromWizard([schedule('s3')], 'cls', {
      s3: { enabled: false },
    });

    expect(storagesOf(spec)[0].pitr).toBeUndefined();
  });

  it('ignores an orphan pitr entry for a storage without schedules', () => {
    const spec = buildBackupSpecFromWizard([schedule('s3')], 'cls', {
      removed: { enabled: true },
    });
    const storages = storagesOf(spec);

    expect(storages).toHaveLength(1);
    expect(storages[0].name).toBe('s3');
    expect(storages[0].pitr).toBeUndefined();
  });
});
