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

import { Backup } from 'shared-types/backups.types';

export const getBackupName = (backup: Backup): string => {
  const metadata = backup.metadata;
  if (metadata && typeof metadata === 'object' && 'name' in metadata) {
    const name = metadata.name;
    if (typeof name === 'string') {
      return name;
    }
  }

  return '';
};

export const getMetadataCreationTimestamp = (
  backup: Backup
): string | undefined => {
  const metadata = backup.metadata;
  if (
    metadata &&
    typeof metadata === 'object' &&
    'creationTimestamp' in metadata
  ) {
    const creationTimestamp = metadata.creationTimestamp;
    if (typeof creationTimestamp === 'string') {
      return creationTimestamp;
    }
  }

  return undefined;
};

export const getSafeTimeValue = (value?: string): number => {
  if (!value) {
    return 0;
  }

  const timeValue = new Date(value).valueOf();
  return Number.isNaN(timeValue) ? 0 : timeValue;
};
