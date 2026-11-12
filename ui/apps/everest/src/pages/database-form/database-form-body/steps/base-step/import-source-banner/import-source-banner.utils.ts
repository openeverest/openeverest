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

import { BackupSourceMetadata } from 'shared-types/backups.types';
import { ImportPrefill } from 'pages/database-form/utils/import-prefill';
import { Messages } from './import-source-banner.messages';

export const getImportSourceAlert = (
  backup: string,
  storage: string,
  sourceMetadata: BackupSourceMetadata | undefined,
  prefill: ImportPrefill
) => {
  const engineVersion = sourceMetadata?.engineVersion;
  const isVersionMissing = !!engineVersion && !prefill.version;
  const versionHint = prefill.version
    ? Messages.versionMatched(prefill.version)
    : engineVersion
      ? Messages.versionMissing(engineVersion)
      : Messages.versionUnknown;

  return {
    severity: isVersionMissing ? ('warning' as const) : ('info' as const),
    message: [
      Messages.source(backup, storage),
      versionHint,
      ...(prefill.topology ? [] : [Messages.topologyUnknown]),
    ].join(' '),
  };
};
