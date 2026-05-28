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

import {
  getSchedulesPayload,
  removeScheduleFromArray,
} from './schedule-form.utils';
import { FlattenedSchedule } from '../schedule-form-dialog-context/schedule-form-dialog-context.types';
import { WizardMode } from 'shared-types/wizard.types';
import { ScheduleFormData } from './schedule-form-schema';
import {
  AmPM,
  TimeValue,
  WeekDays,
} from '../../time-selection/time-selection.types';

const makeSchedule = (
  overrides: Partial<FlattenedSchedule> = {}
): FlattenedSchedule => ({
  enabled: true,
  name: 'schedule-1',
  cron: '0 0 * * *',
  storageName: 'storage-a',
  retentionCopies: 3,
  ...overrides,
});

const makeFormData = (
  overrides: Partial<ScheduleFormData> = {}
): ScheduleFormData => ({
  scheduleName: 'new-schedule',
  backupClassName: 'percona-backup-mongodb',
  storageLocation: { metadata: { name: 'storage-b' } },
  retentionCopies: '5',
  selectedTime: TimeValue.days,
  minute: 30,
  hour: 2,
  amPm: AmPM.AM,
  weekDay: WeekDays.Mo,
  onDay: 1,
  ...overrides,
});

describe('getSchedulesPayload', () => {
  describe('New mode', () => {
    it('appends new schedule to existing array', () => {
      const existing = [makeSchedule({ name: 'existing' })];
      const result = getSchedulesPayload({
        formData: makeFormData(),
        mode: WizardMode.New,
        schedules: existing,
      });
      expect(result).toHaveLength(2);
      expect(result[0].name).toBe('existing');
      expect(result[1].name).toBe('new-schedule');
    });

    it('creates schedule from empty array', () => {
      const result = getSchedulesPayload({
        formData: makeFormData(),
        mode: WizardMode.New,
        schedules: [],
      });
      expect(result).toHaveLength(1);
      expect(result[0].name).toBe('new-schedule');
      expect(result[0].storageName).toBe('storage-b');
      expect(result[0].retentionCopies).toBe(5);
      expect(result[0].enabled).toBe(true);
    });

    it('extracts storage name from object storageLocation', () => {
      const result = getSchedulesPayload({
        formData: makeFormData({
          storageLocation: { metadata: { name: 'my-storage' } },
        }),
        mode: WizardMode.New,
        schedules: [],
      });
      expect(result[0].storageName).toBe('my-storage');
    });

    it('extracts storage name from string storageLocation', () => {
      const result = getSchedulesPayload({
        formData: makeFormData({
          storageLocation:
            'string-storage' as unknown as ScheduleFormData['storageLocation'],
        }),
        mode: WizardMode.New,
        schedules: [],
      });
      expect(result[0].storageName).toBe('string-storage');
    });

    it('generates cron expression from time selection fields', () => {
      const result = getSchedulesPayload({
        formData: makeFormData({
          selectedTime: TimeValue.days,
          hour: 3,
          minute: 15,
          amPm: AmPM.PM,
        }),
        mode: WizardMode.New,
        schedules: [],
      });
      // 3 PM = 15:00 in 24h, so cron should be "15 15 * * *"
      expect(result[0].cron).toBe('15 15 * * *');
    });

    it('includes dynamic config fields when present', () => {
      const result = getSchedulesPayload({
        formData: {
          ...makeFormData(),
          config: { compressionType: 'gzip', level: '5' },
        } as ScheduleFormData,
        mode: WizardMode.New,
        schedules: [],
      });
      expect(result[0].config).toEqual({ compressionType: 'gzip', level: '5' });
    });

    it('omits config when no dynamic fields present', () => {
      const result = getSchedulesPayload({
        formData: makeFormData(),
        mode: WizardMode.New,
        schedules: [],
      });
      expect(result[0]).not.toHaveProperty('config');
    });
  });

  describe('Edit mode', () => {
    it('replaces existing schedule by name', () => {
      const existing = [
        makeSchedule({ name: 'target', storageName: 'old-storage' }),
        makeSchedule({ name: 'other', storageName: 'keep-me' }),
      ];
      const result = getSchedulesPayload({
        formData: makeFormData({ scheduleName: 'target' }),
        mode: WizardMode.Edit,
        schedules: existing,
      });
      expect(result).toHaveLength(2);
      expect(result[0].name).toBe('target');
      expect(result[0].storageName).toBe('storage-b');
      expect(result[1].name).toBe('other');
      expect(result[1].storageName).toBe('keep-me');
    });

    it('preserves array when schedule name not found', () => {
      const existing = [makeSchedule({ name: 'other' })];
      const result = getSchedulesPayload({
        formData: makeFormData({ scheduleName: 'nonexistent' }),
        mode: WizardMode.Edit,
        schedules: existing,
      });
      expect(result).toHaveLength(1);
      expect(result[0].name).toBe('other');
    });

    it('does not append a new entry', () => {
      const existing = [makeSchedule({ name: 'only' })];
      const result = getSchedulesPayload({
        formData: makeFormData({ scheduleName: 'only' }),
        mode: WizardMode.Edit,
        schedules: existing,
      });
      expect(result).toHaveLength(1);
    });
  });

  describe('Import mode (no-op)', () => {
    it('returns original schedules unchanged', () => {
      const existing = [makeSchedule()];
      const result = getSchedulesPayload({
        formData: makeFormData(),
        mode: WizardMode.Import,
        schedules: existing,
      });
      expect(result).toEqual(existing);
    });
  });
});

describe('removeScheduleFromArray', () => {
  it('removes schedule by name', () => {
    const schedules = [
      makeSchedule({ name: 'keep' }),
      makeSchedule({ name: 'remove-me' }),
      makeSchedule({ name: 'also-keep' }),
    ];
    const result = removeScheduleFromArray('remove-me', schedules);
    expect(result).toHaveLength(2);
    expect(result.map((s) => s.name)).toEqual(['keep', 'also-keep']);
  });

  it('returns empty array when last schedule removed', () => {
    const schedules = [makeSchedule({ name: 'only' })];
    const result = removeScheduleFromArray('only', schedules);
    expect(result).toHaveLength(0);
  });

  it('returns original array when name not found', () => {
    const schedules = [makeSchedule({ name: 'existing' })];
    const result = removeScheduleFromArray('nonexistent', schedules);
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('existing');
  });

  it('handles empty array', () => {
    const result = removeScheduleFromArray('any', []);
    expect(result).toHaveLength(0);
  });
});
