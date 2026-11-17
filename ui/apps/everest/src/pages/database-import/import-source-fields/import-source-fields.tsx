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

import { Alert, Box, FormGroup, MenuItem } from '@mui/material';
import {
  ActionableLabeledContent,
  AutoCompleteInput,
  SelectInput,
} from '@percona/ui-lib';
import { useFormContext } from 'react-hook-form';
import BackupsActionableAlert from 'components/actionable-alert/backups-actionable-alert';
import BackupStoragesInput from 'components/backup-storages-input';
import { useAddBackupStorage } from 'hooks/api/backup-storages/useAddBackupStorage';
import { useExtensionCatalog } from 'hooks/api/extension-catalog';
import { CreateEditModalStorage } from 'pages/settings/storage-locations/createEditModal/create-edit-modal';
import { Provider } from 'shared-types/api.types';
import { BackupClass } from 'shared-types/backups.types';
import { BackupStorageCRD } from 'shared-types/backupStorages.types';
import { Messages } from '../database-import.messages';
import {
  DatabaseImportFields,
  DatabaseImportFormValues,
} from '../database-import-schema';

interface ImportSourceFieldsProps {
  namespace: string;
  namespaces: string[];
  isNamespacesLoading: boolean;
  storages: BackupStorageCRD[];
  isStoragesLoading: boolean;
  importClasses: BackupClass[];
  selectedClass?: BackupClass;
  classProviders: Provider[];
  onStorageAdded: (storageName: string) => void;
}

export const ImportSourceFields = ({
  namespace,
  namespaces,
  isNamespacesLoading,
  storages,
  isStoragesLoading,
  importClasses,
  selectedClass,
  classProviders,
  onStorageAdded,
}: ImportSourceFieldsProps) => {
  const { setValue } = useFormContext<DatabaseImportFormValues>();
  const { getProviderMeta } = useExtensionCatalog();
  const addStorage = useAddBackupStorage(namespace, onStorageAdded);
  const hasNoStorages =
    !!namespace && !isStoragesLoading && storages.length === 0;

  return (
    <FormGroup sx={{ mt: 3 }}>
      <AutoCompleteInput
        name={DatabaseImportFields.namespace}
        label={Messages.labels.namespace}
        loading={isNamespacesLoading}
        options={namespaces}
        onChange={() => setValue(DatabaseImportFields.storage, null)}
        autoCompleteProps={{
          disableClearable: true,
          isOptionEqualToValue: (option, value) => option === value,
        }}
      />
      {hasNoStorages ? (
        <Box sx={{ mt: 3 }}>
          <BackupsActionableAlert
            namespace={namespace}
            message={Messages.noStorages}
          />
        </Box>
      ) : (
        !!namespace && (
          // The bucket with the backups is often not added to the namespace yet.
          <ActionableLabeledContent
            actionButtonProps={
              addStorage.canAdd
                ? {
                    buttonText: Messages.addStorage,
                    dataTestId: 'add-storage-button',
                    onClick: addStorage.open,
                  }
                : undefined
            }
          >
            <BackupStoragesInput
              name={DatabaseImportFields.storage}
              namespace={namespace}
              schedules={[]}
            />
          </ActionableLabeledContent>
        )
      )}
      {addStorage.isOpen && (
        <CreateEditModalStorage {...addStorage.modalProps} />
      )}
      <SelectInput
        name={DatabaseImportFields.backupClassName}
        label={Messages.labels.backupClass}
        selectFieldProps={{ label: Messages.labels.backupClass }}
      >
        {importClasses.map((bc) => (
          <MenuItem key={bc.metadata?.name} value={bc.metadata?.name ?? ''}>
            {bc.spec.displayName || bc.metadata?.name}
          </MenuItem>
        ))}
      </SelectInput>
      {classProviders.length > 1 && (
        <SelectInput
          name={DatabaseImportFields.providerName}
          label={Messages.labels.provider}
          selectFieldProps={{ label: Messages.labels.provider }}
        >
          {classProviders.map((provider) => {
            const name = provider.metadata?.name ?? '';
            return (
              <MenuItem key={name} value={name}>
                {getProviderMeta(name)?.displayName ?? name}
              </MenuItem>
            );
          })}
        </SelectInput>
      )}
      {selectedClass && classProviders.length === 0 && (
        <Alert severity="warning" sx={{ mt: 3 }}>
          {Messages.providerNotInstalled}
        </Alert>
      )}
    </FormGroup>
  );
};
