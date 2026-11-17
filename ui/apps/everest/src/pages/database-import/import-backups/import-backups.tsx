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

import { Alert, Box, Button } from '@mui/material';
import { Table } from '@percona/ui-lib';
import { format } from 'date-fns';
import {
  MRT_ColumnDef,
  MRT_RowSelectionState,
  MRT_Updater,
} from 'material-react-table';
import { useController, useFormContext } from 'react-hook-form';
import { DATE_FORMAT } from 'consts';
import { IMPORT_BACKUPS_TABLE_NAME } from '../database-import.constants';
import { Messages } from '../database-import.messages';
import {
  DatabaseImportFields,
  DatabaseImportFormValues,
} from '../database-import-schema';
import { ImportBackupRow, ScanTarget } from '../database-import.types';
import { getScanAlert } from '../database-import.utils';
import { useBackupScan } from '../hooks/use-backup-scan';

interface ImportBackupsProps extends ScanTarget {
  namespace: string;
  rows: ImportBackupRow[];
  isLoading: boolean;
}

const columns: MRT_ColumnDef<ImportBackupRow>[] = [
  {
    accessorKey: 'completedAt',
    header: Messages.backups.completedAt,
    Cell: ({ cell }) => format(new Date(cell.getValue<string>()), DATE_FORMAT),
  },
  {
    accessorKey: 'path',
    header: Messages.backups.path,
    size: 280,
  },
  {
    accessorKey: 'sourceMetadata.engineVersion',
    header: Messages.backups.version,
    size: 120,
  },
  {
    accessorKey: 'sourceMetadata.backupType',
    header: Messages.backups.type,
    size: 120,
  },
  {
    accessorKey: 'sourceMetadata.topology',
    header: Messages.backups.topology,
    size: 120,
  },
];

export const ImportBackups = ({
  namespace,
  storageName,
  backupClassName,
  rows,
  isLoading,
}: ImportBackupsProps) => {
  const { control } = useFormContext<DatabaseImportFormValues>();
  const { field } = useController({
    control,
    name: DatabaseImportFields.backupName,
  });
  const { scan, canScan, hasScanned, ...scanProgress } = useBackupScan({
    namespace,
    storageName,
    backupClassName,
  });

  const scanAlert = getScanAlert(scanProgress);
  const rowSelection: MRT_RowSelectionState = field.value
    ? { [field.value]: true }
    : {};

  const handleRowSelectionChange = (
    updater: MRT_Updater<MRT_RowSelectionState>
  ) => {
    const next =
      typeof updater === 'function' ? updater(rowSelection) : updater;
    field.onChange(Object.keys(next).find((id) => next[id]) ?? '');
  };

  return (
    <Box sx={{ mt: 3 }}>
      {scanAlert && (
        <Alert severity={scanAlert.severity} sx={{ mb: 2 }}>
          {scanAlert.text}
        </Alert>
      )}
      <Table
        tableName={IMPORT_BACKUPS_TABLE_NAME}
        columns={columns}
        data={rows}
        state={{
          isLoading,
          rowSelection,
          showProgressBars: scanProgress.isScanning,
        }}
        getRowId={(row) => row.name}
        enableRowSelection
        enableMultiRowSelection={false}
        positionToolbarAlertBanner="none"
        onRowSelectionChange={handleRowSelectionChange}
        enableRowHoverAction
        rowHoverAction={(row) => field.onChange(row.id)}
        noDataMessage={
          scanProgress.isScanning && !canScan
            ? Messages.scan.running
            : Messages.backups.noBackups
        }
        renderTopToolbarCustomActions={() => (
          <Button
            variant="outlined"
            size="small"
            disabled={!canScan}
            onClick={scan}
            data-testid="scan-storage-button"
          >
            {!canScan
              ? Messages.backups.scanning
              : hasScanned
                ? Messages.backups.scanAgain
                : Messages.backups.scan}
          </Button>
        )}
      />
    </Box>
  );
};
