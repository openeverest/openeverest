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

import { schema } from './on-demand-backup-modal.types';

describe('on-demand-backup schema', () => {
  const existingBackups = ['backup-abc123', 'backup-xyz456'];
  const testSchema = schema(existingBackups);

  describe('duplicate name rejection', () => {
    it('rejects a name that already exists in the backups list', () => {
      const result = testSchema.safeParse({
        name: 'backup-abc123',
        backupClassName: 'standard',
        storageName: { name: 'my-storage' },
      });
      expect(result.success).toBe(false);
      if (!result.success) {
        const nameErrors = result.error.issues.filter(
          (i) => i.path[0] === 'name'
        );
        expect(nameErrors.length).toBeGreaterThan(0);
        expect(nameErrors[0].message).toBe(
          'You already have a backup with this name'
        );
      }
    });

    it('accepts a unique name', () => {
      const result = testSchema.safeParse({
        name: 'backup-new001',
        backupClassName: 'standard',
        storageName: { name: 'my-storage' },
      });
      expect(result.success).toBe(true);
    });
  });
});
