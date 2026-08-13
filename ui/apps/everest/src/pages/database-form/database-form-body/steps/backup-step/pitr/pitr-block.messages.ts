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

export const Messages = {
  title: 'Point-in-time Recovery',
  description:
    'Point-in-time recovery keeps continuous backups of your database so you can recover from accidental writes or deletes.',
  listLeadIn: (maxEnabled?: number) => {
    if (maxEnabled === 1) {
      return 'Restore your database to any point in time by enabling PITR on a backup storage.';
    }
    if (maxEnabled != null) {
      return `Restore your database to any point in time by enabling PITR on up to ${maxEnabled} backup storages.`;
    }
    return 'Restore your database to any point in time by enabling PITR on one or more backup storages.';
  },
  limitReached: (max: number) =>
    `Maximum ${max} PITR-enabled storage${max > 1 ? 's' : ''} for this provider.`,
};
