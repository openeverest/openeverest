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

// Mock BackupClass uiSchema data for development.
// TODO: Remove this file once PR #2226 is merged and uiSchema comes from the API.

import type { Section } from 'components/ui-generator/ui-generator.types';
import { FieldType } from 'components/ui-generator/ui-generator.types';

export type BackupClassUiSchemaSections = {
  backup?: Section;
  pitr?: Section;
  restore?: Section;
};

/**
 * PSMDB BackupClass uiSchema mock.
 * Matches the shape described in docs/proposals/backups-restore-architecture.md.
 */
export const psmdbBackupClassUiSchema: BackupClassUiSchemaSections = {
  backup: {
    label: 'Backup Configuration',
    componentsOrder: ['type', 'compressionType', 'compressionLevel'],
    components: {
      type: {
        uiType: FieldType.Select,
        path: 'type',
        fieldParams: {
          label: 'Backup Type',
          options: [
            { label: 'Logical', value: 'logical' },
            { label: 'Physical', value: 'physical' },
          ],
          defaultValue: 'logical',
        },
        validation: { required: true },
      },
      compressionType: {
        uiType: FieldType.Select,
        path: 'compressionType',
        fieldParams: {
          label: 'Compression',
          options: [
            { label: 'None', value: 'none' },
            { label: 'Gzip', value: 'gzip' },
            { label: 'Snappy', value: 'snappy' },
            { label: 'LZ4', value: 'lz4' },
            { label: 'Zstandard', value: 'zstd' },
          ],
          defaultValue: 'snappy',
        },
      },
      compressionLevel: {
        uiType: FieldType.Number,
        path: 'compressionLevel',
        fieldParams: {
          label: 'Compression Level',
          defaultValue: 6,
        },
        validation: { min: 0, max: 22 },
      },
    },
  },
  pitr: {
    label: 'PITR Configuration',
    componentsOrder: ['oplogSpanMin', 'compressionType'],
    components: {
      oplogSpanMin: {
        uiType: FieldType.Number,
        path: 'oplogSpanMin',
        fieldParams: {
          label: 'Oplog Span (minutes)',
          tooltip: 'Interval between oplog chunk boundaries',
          defaultValue: 10,
        },
        validation: { required: true, min: 1 },
      },
      compressionType: {
        uiType: FieldType.Select,
        path: 'compressionType',
        fieldParams: {
          label: 'Oplog Compression',
          options: [
            { label: 'None', value: 'none' },
            { label: 'Snappy', value: 'snappy' },
            { label: 'Zstandard', value: 'zstd' },
          ],
          defaultValue: 'snappy',
        },
      },
    },
  },
  restore: {
    label: 'Restore Configuration',
    components: {},
  },
};

/**
 * Map of provider name → mock uiSchema sections.
 * Used to resolve uiSchema by provider until PR #2226 delivers the real data via API.
 */
export const mockUiSchemaByProvider: Record<
  string,
  BackupClassUiSchemaSections
> = {
  'percona-server-mongodb': psmdbBackupClassUiSchema,
};
