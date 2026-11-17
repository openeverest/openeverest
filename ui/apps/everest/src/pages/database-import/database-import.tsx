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

import { useMemo } from 'react';
import { Alert } from '@mui/material';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import { useNamespaceBackups } from 'hooks/api/backups/useBackups';
import { useClusterName } from 'hooks/api/useClusterName';
import DatabaseFormStepControllers from 'pages/database-form/database-form-body/database-form-step-controllers';
import { StepHeader } from 'pages/database-form/database-form-body/steps-old/step-header/step-header';
import { buildImportNavigationRouteState } from 'pages/database-form/import-navigation-state';
import { useImportNavigationState } from 'pages/database-form/hooks/use-import-navigation-state';
import { Messages } from './database-import.messages';
import {
  DatabaseImportFields,
  DatabaseImportFormValues,
  databaseImportSchema,
} from './database-import-schema';
import {
  getImportBackupRows,
  getImportFormDefaults,
} from './database-import.utils';
import { useImportSource } from './hooks/use-import-source';
import { ImportBackups } from './import-backups';
import { ImportSourceFields } from './import-source-fields';

export const DatabaseImportPage = () => {
  const navigate = useNavigate();
  const clusterName = useClusterName();
  const previousNavigation = useImportNavigationState();
  const methods = useForm<DatabaseImportFormValues>({
    mode: 'onChange',
    resolver: zodResolver(databaseImportSchema),
    defaultValues: getImportFormDefaults(previousNavigation),
  });
  const {
    namespace,
    storageName,
    backupClassName,
    namespaces,
    isNamespacesLoading,
    storages,
    isStoragesLoading,
    importClasses,
    isClassesLoading,
    selectedClass,
    classProviders,
    selectStorage,
  } = useImportSource(
    methods,
    previousNavigation?.importSource.dataSource.storageName
  );
  const hasScanTarget = !!storageName && !!backupClassName;

  const { data: backups = [], isLoading: isBackupsLoading } =
    useNamespaceBackups(
      clusterName,
      namespace,
      {
        backupStorage: storageName,
        backupClass: backupClassName,
        backupOrigin: 'External',
      },
      { enabled: hasScanTarget, refetchOnWindowFocus: true }
    );
  const rows = useMemo(() => getImportBackupRows(backups), [backups]);
  const backupName = useWatch({
    control: methods.control,
    name: DatabaseImportFields.backupName,
  });
  // A pick made for another storage or class stays, but can't be imported.
  const isBackupListed = rows.some((row) => row.name === backupName);

  const unavailableMessage =
    !isNamespacesLoading && namespaces.length === 0
      ? Messages.noNamespaces
      : !isClassesLoading && importClasses.length === 0
        ? Messages.noImportClasses
        : undefined;

  const handleContinue = methods.handleSubmit((values) => {
    const provider = classProviders.find(
      (p) => p.metadata?.name === values.providerName
    );
    const selectedStorageName = values.storage?.metadata?.name;
    const backup = rows.find((row) => row.name === values.backupName);
    if (!provider || !selectedStorageName || !backup) {
      return;
    }
    navigate('/databases/new', {
      state: buildImportNavigationRouteState(provider, {
        namespace: values.namespace,
        backupClassName: values.backupClassName,
        dataSource: {
          type: 'Backup',
          backupName: values.backupName,
          storageName: selectedStorageName,
        },
        sourceMetadata: backup.sourceMetadata,
      }),
    });
  });

  return (
    <FormProvider {...methods}>
      <form onSubmit={handleContinue}>
        <StepHeader
          pageTitle={Messages.pageTitle}
          pageDescription={Messages.pageDescription}
        />
        {unavailableMessage ? (
          <Alert severity="info">{unavailableMessage}</Alert>
        ) : (
          <>
            <ImportSourceFields
              namespace={namespace}
              namespaces={namespaces}
              isNamespacesLoading={isNamespacesLoading}
              storages={storages}
              isStoragesLoading={isStoragesLoading}
              importClasses={importClasses}
              selectedClass={selectedClass}
              classProviders={classProviders}
              onStorageAdded={selectStorage}
            />
            {hasScanTarget && (
              <ImportBackups
                namespace={namespace}
                storageName={storageName}
                backupClassName={backupClassName}
                rows={rows}
                isLoading={isBackupsLoading}
              />
            )}
          </>
        )}
        <DatabaseFormStepControllers
          disableBack
          disableNext={!methods.formState.isValid || !isBackupListed}
          onNextClick={handleContinue}
          onCancel={() => navigate('/databases')}
        />
      </form>
    </FormProvider>
  );
};
