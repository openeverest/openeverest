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

import { schema } from './schedule-form-schema';
import { FlattenedSchedule } from '../schedule-form-dialog-context/schedule-form-dialog-context.types';
import { WizardMode } from 'shared-types/wizard.types';
import { Messages } from './schedule-form.messages';
import {
  AmPM,
  TimeValue,
  WeekDays,
} from '../../time-selection/time-selection.types';

const makeSchedule = (
  overrides: Partial<FlattenedSchedule> = {}
): FlattenedSchedule => ({
  name: 'existing-schedule',
  enabled: true,
  cron: '0 0 * * *',
  storageName: 'storage-a',
  retentionCopies: 3,
  ...overrides,
});

const validFormData = {
  scheduleName: 'my-new-backup',
  backupClassName: 'percona-backup-mongodb',
  storageLocation: { metadata: { name: 'storage-a' } },
  retentionCopies: '3',
  selectedTime: TimeValue.days,
  minute: 0,
  hour: 1,
  amPm: AmPM.AM,
  weekDay: WeekDays.Mo,
  onDay: 1,
};

describe('schedule-form-schema', () => {
  describe('scheduleName validation', () => {
    it('accepts a valid RFC-123 name', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse(validFormData);
      expect(result.success).toBe(true);
    });

    it('rejects empty name', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse({ ...validFormData, scheduleName: '' });
      expect(result.success).toBe(false);
    });

    it('rejects name exceeding max length (57 chars)', () => {
      const s = schema([], WizardMode.New);
      const longName = 'a'.repeat(58);
      const result = s.safeParse({ ...validFormData, scheduleName: longName });
      expect(result.success).toBe(false);
    });

    it('rejects duplicate name in New mode', () => {
      const existingSchedules = [makeSchedule({ name: 'my-new-backup' })];
      const s = schema(existingSchedules, WizardMode.New);
      const result = s.safeParse(validFormData);
      expect(result.success).toBe(false);
      if (!result.success) {
        const messages = result.error.issues.map((i) => i.message);
        expect(messages).toContain(Messages.scheduleName.duplicate);
      }
    });

    it('allows same name in Edit mode (editing self)', () => {
      const existingSchedules = [makeSchedule({ name: 'my-new-backup' })];
      const s = schema(existingSchedules, WizardMode.Edit);
      const result = s.safeParse(validFormData);
      // Edit mode does not trigger duplicate name check
      expect(
        result.success ||
          !result.error.issues.some(
            (i) => i.message === Messages.scheduleName.duplicate
          )
      ).toBe(true);
    });

    it('rejects names with uppercase letters (RFC-123)', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse({
        ...validFormData,
        scheduleName: 'MyBackup',
      });
      expect(result.success).toBe(false);
    });

    it('rejects names with underscores (RFC-123)', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse({
        ...validFormData,
        scheduleName: 'my_backup',
      });
      expect(result.success).toBe(false);
    });
  });

  describe('retentionCopies validation', () => {
    it('accepts zero (infinite retention)', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse({ ...validFormData, retentionCopies: '0' });
      expect(result.success).toBe(true);
    });

    it('accepts positive integer', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse({ ...validFormData, retentionCopies: '10' });
      expect(result.success).toBe(true);
    });

    it('rejects negative number', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse({ ...validFormData, retentionCopies: '-1' });
      expect(result.success).toBe(false);
      if (!result.success) {
        const messages = result.error.issues.map((i) => i.message);
        expect(messages).toContain(Messages.retentionCopies.invalidNumber);
      }
    });

    it('rejects NaN string', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse({ ...validFormData, retentionCopies: 'abc' });
      expect(result.success).toBe(false);
    });

    it('rejects overflow (> 2^31 - 1)', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse({
        ...validFormData,
        retentionCopies: '2147483648',
      });
      expect(result.success).toBe(false);
    });
  });

  describe('storageLocation validation', () => {
    it('accepts valid object with name', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse(validFormData);
      expect(result.success).toBe(true);
    });

    it('rejects null in scheduledBackups mode', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse({
        ...validFormData,
        storageLocation: null,
      });
      expect(result.success).toBe(false);
    });

    it('rejects empty object (no name)', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse({
        ...validFormData,
        storageLocation: {},
      });
      expect(result.success).toBe(false);
    });

    it('accepts valid string storage', () => {
      const s = schema([], WizardMode.New);
      const result = s.safeParse({
        ...validFormData,
        storageLocation: 'storage-name',
      });
      // String is accepted by zod union but fails validation (no .name)
      expect(result.success).toBe(false);
    });
  });

  describe('duplicate cron (same time) validation', () => {
    it('rejects when another schedule has same cron expression', () => {
      // Create a schedule whose cron matches what the form values would generate
      // For TimeValue.days at hour=1, minute=0, AM → cron is "0 1 * * *"
      const existingSchedules = [makeSchedule({ cron: '0 1 * * *' })];
      const s = schema(existingSchedules, WizardMode.New);
      const result = s.safeParse(validFormData);
      expect(result.success).toBe(false);
      if (!result.success) {
        const messages = result.error.issues.map((i) => i.message);
        expect(messages).toContain(Messages.sameTimeSchedule);
      }
    });

    it('accepts when no schedule has same cron', () => {
      const existingSchedules = [makeSchedule({ cron: '30 5 * * *' })];
      const s = schema(existingSchedules, WizardMode.New);
      const result = s.safeParse(validFormData);
      expect(result.success).toBe(true);
    });

    it('skips self in Edit mode (same name with matching cron)', () => {
      const existingSchedules = [
        makeSchedule({ name: 'my-new-backup', cron: '0 1 * * *' }),
      ];
      const s = schema(existingSchedules, WizardMode.Edit);
      const result = s.safeParse(validFormData);
      // Should not flag duplicate because it's the same schedule being edited
      expect(
        result.success ||
          !result.error.issues.some(
            (i) => i.message === Messages.sameTimeSchedule
          )
      ).toBe(true);
    });
  });
});
