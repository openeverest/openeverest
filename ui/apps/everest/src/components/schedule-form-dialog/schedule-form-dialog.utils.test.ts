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

import { WizardMode } from 'shared-types/wizard.types';
import { FlattenedSchedule } from './schedule-form-dialog-context/schedule-form-dialog-context.types';
import {
  sameScheduleFunc,
  sameStorageLocationFunc,
  scheduleModalDefaultValues,
} from './schedule-form-dialog.utils';
import { ScheduleFormFields } from './schedule-form/schedule-form.types';

const makeSchedule = (
  overrides: Partial<FlattenedSchedule> = {}
): FlattenedSchedule => ({
  name: 'schedule-1',
  enabled: true,
  cron: '0 0 * * *',
  storageName: 'storage-a',
  retentionCopies: 3,
  ...overrides,
});

describe('sameScheduleFunc', () => {
  const schedules: FlattenedSchedule[] = [
    makeSchedule({ name: 'daily', cron: '0 0 * * *' }),
    makeSchedule({ name: 'hourly', cron: '0 * * * *' }),
    makeSchedule({ name: 'weekly', cron: '0 0 * * 1' }),
  ];

  describe('New mode', () => {
    it('returns match when duplicate cron exists', () => {
      const result = sameScheduleFunc(
        schedules,
        WizardMode.New,
        '0 0 * * *',
        'new-schedule'
      );
      expect(result).toBeDefined();
      expect(result?.name).toBe('daily');
    });

    it('returns undefined when no duplicate cron', () => {
      const result = sameScheduleFunc(
        schedules,
        WizardMode.New,
        '30 2 * * *',
        'new-schedule'
      );
      expect(result).toBeUndefined();
    });

    it('returns match even if cron matches the same name (New mode does not skip self)', () => {
      const result = sameScheduleFunc(
        schedules,
        WizardMode.New,
        '0 0 * * *',
        'daily'
      );
      expect(result).toBeDefined();
    });
  });

  describe('Edit mode', () => {
    it('skips self (same name) when cron matches', () => {
      const result = sameScheduleFunc(
        schedules,
        WizardMode.Edit,
        '0 0 * * *',
        'daily'
      );
      expect(result).toBeUndefined();
    });

    it('returns match when a different schedule has the same cron', () => {
      const result = sameScheduleFunc(
        schedules,
        WizardMode.Edit,
        '0 0 * * *',
        'hourly'
      );
      expect(result).toBeDefined();
      expect(result?.name).toBe('daily');
    });

    it('returns undefined when no other schedule has the same cron', () => {
      const result = sameScheduleFunc(
        schedules,
        WizardMode.Edit,
        '30 5 * * *',
        'daily'
      );
      expect(result).toBeUndefined();
    });
  });

  it('returns undefined for empty schedules list', () => {
    const result = sameScheduleFunc([], WizardMode.New, '0 0 * * *', 'any');
    expect(result).toBeUndefined();
  });
});

describe('sameStorageLocationFunc', () => {
  const schedules: FlattenedSchedule[] = [
    makeSchedule({ name: 'sched-1', storageName: 'storage-a' }),
    makeSchedule({ name: 'sched-2', storageName: 'storage-b' }),
  ];

  describe('New mode', () => {
    it('finds duplicate when string storage name matches', () => {
      const result = sameStorageLocationFunc(
        schedules,
        WizardMode.New,
        'storage-a',
        'new-sched'
      );
      expect(result).toBeDefined();
      expect(result?.name).toBe('sched-1');
    });

    it('finds duplicate when object { name } storage matches', () => {
      const result = sameStorageLocationFunc(
        schedules,
        WizardMode.New,
        { name: 'storage-b' },
        'new-sched'
      );
      expect(result).toBeDefined();
      expect(result?.name).toBe('sched-2');
    });

    it('returns undefined for null storage', () => {
      const result = sameStorageLocationFunc(
        schedules,
        WizardMode.New,
        null,
        'new-sched'
      );
      expect(result).toBeUndefined();
    });

    it('returns undefined for undefined storage', () => {
      const result = sameStorageLocationFunc(
        schedules,
        WizardMode.New,
        undefined,
        'new-sched'
      );
      expect(result).toBeUndefined();
    });

    it('returns undefined when no duplicate storage', () => {
      const result = sameStorageLocationFunc(
        schedules,
        WizardMode.New,
        'storage-c',
        'new-sched'
      );
      expect(result).toBeUndefined();
    });
  });

  describe('Edit mode', () => {
    it('skips self (same name) when storage matches', () => {
      const result = sameStorageLocationFunc(
        schedules,
        WizardMode.Edit,
        'storage-a',
        'sched-1'
      );
      expect(result).toBeUndefined();
    });

    it('returns match when different schedule uses same storage', () => {
      const result = sameStorageLocationFunc(
        schedules,
        WizardMode.Edit,
        'storage-a',
        'sched-2'
      );
      expect(result).toBeDefined();
      expect(result?.name).toBe('sched-1');
    });
  });

  it('returns undefined for empty schedules list', () => {
    const result = sameStorageLocationFunc(
      [],
      WizardMode.New,
      'storage-a',
      'any'
    );
    expect(result).toBeUndefined();
  });
});

describe('scheduleModalDefaultValues', () => {
  describe('New mode', () => {
    it('returns defaults with generated name prefix', () => {
      const result = scheduleModalDefaultValues(WizardMode.New);
      expect(result[ScheduleFormFields.scheduleName]).toMatch(/^backup-/);
      expect(result[ScheduleFormFields.storageLocation]).toBeNull();
      expect(result[ScheduleFormFields.retentionCopies]).toBe('0');
    });

    it('includes time selection defaults', () => {
      const result = scheduleModalDefaultValues(WizardMode.New);
      expect(result).toHaveProperty('selectedTime');
      expect(result).toHaveProperty('minute');
      expect(result).toHaveProperty('hour');
    });
  });

  describe('Edit mode', () => {
    const selectedSchedule = makeSchedule({
      name: 'my-schedule',
      storageName: 'my-storage',
      cron: '0 12 * * *',
      retentionCopies: 5,
      config: { compressionType: 'gzip' },
    });

    it('populates name from selected schedule', () => {
      const result = scheduleModalDefaultValues(
        WizardMode.Edit,
        selectedSchedule
      );
      expect(result[ScheduleFormFields.scheduleName]).toBe('my-schedule');
    });

    it('populates storage location as object', () => {
      const result = scheduleModalDefaultValues(
        WizardMode.Edit,
        selectedSchedule
      );
      expect(result[ScheduleFormFields.storageLocation]).toEqual({
        metadata: { name: 'my-storage' },
      });
    });

    it('populates retention copies as string', () => {
      const result = scheduleModalDefaultValues(
        WizardMode.Edit,
        selectedSchedule
      );
      expect(result[ScheduleFormFields.retentionCopies]).toBe('5');
    });

    it('includes config when present', () => {
      const result = scheduleModalDefaultValues(
        WizardMode.Edit,
        selectedSchedule
      );
      expect(result).toHaveProperty('config', { compressionType: 'gzip' });
    });

    it('omits config when not present', () => {
      const scheduleWithoutConfig = makeSchedule({
        name: 'no-config',
        storageName: 'storage-x',
        cron: '0 6 * * *',
      });
      const result = scheduleModalDefaultValues(
        WizardMode.Edit,
        scheduleWithoutConfig
      );
      expect(result).not.toHaveProperty('config');
    });

    it('falls back to New mode defaults when selectedSchedule is undefined', () => {
      const result = scheduleModalDefaultValues(WizardMode.Edit, undefined);
      expect(result[ScheduleFormFields.scheduleName]).toMatch(/^backup-/);
      expect(result[ScheduleFormFields.storageLocation]).toBeNull();
    });
  });
});
