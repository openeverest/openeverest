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

import { useEffect, useMemo, useState } from 'react';
import { UseFormReturn, useWatch } from 'react-hook-form';
import { useImportBackupClasses } from 'hooks/api/backup-classes/useBackupClasses';
import { useImportNamespaces } from 'hooks/api/backup-imports';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages';
import { useProviders } from 'hooks/api/providers/useProviders';
import { useClusterName } from 'hooks/api/useClusterName';
import {
  DatabaseImportFields,
  DatabaseImportFormValues,
} from '../database-import-schema';
import { getClassProviders } from '../database-import.utils';

// Options for the import source, preselecting the first one of each so the
// common single-option case needs no input.
export const useImportSource = (
  {
    control,
    setValue,
  }: Pick<UseFormReturn<DatabaseImportFormValues>, 'control' | 'setValue'>,
  initialStorageName?: string
) => {
  const clusterName = useClusterName();
  const namespace = useWatch({ control, name: DatabaseImportFields.namespace });
  const storage = useWatch({ control, name: DatabaseImportFields.storage });
  const backupClassName = useWatch({
    control,
    name: DatabaseImportFields.backupClassName,
  });
  const providerName = useWatch({
    control,
    name: DatabaseImportFields.providerName,
  });
  const storageName = storage?.metadata?.name ?? '';

  const { namespaces, isLoading: isNamespacesLoading } = useImportNamespaces();
  const { data: storages = [], isLoading: isStoragesLoading } =
    useBackupStoragesByNamespace(namespace, { enabled: !!namespace });
  const { data: importClasses, isLoading: isClassesLoading } =
    useImportBackupClasses(clusterName);
  const { data: providers = [], isLoading: isProvidersLoading } =
    useProviders();

  const selectedClass = importClasses.find(
    (bc) => bc.metadata?.name === backupClassName
  );
  const classProviders = useMemo(
    () => getClassProviders(selectedClass, providers),
    [selectedClass, providers]
  );

  // The storage input holds the storage object, so a storage known only by
  // name (restored from the wizard, or just added) is picked once it's listed.
  const [storageNameToSelect, selectStorage] = useState(initialStorageName);
  useEffect(() => {
    const listed = storages.find(
      (s) => s.metadata?.name === storageNameToSelect
    );
    if (listed) {
      setValue(DatabaseImportFields.storage, listed, { shouldValidate: true });
      selectStorage(undefined);
    }
  }, [storages, storageNameToSelect, setValue]);

  useEffect(() => {
    if (!namespace && namespaces.length > 0) {
      setValue(DatabaseImportFields.namespace, namespaces[0], {
        shouldValidate: true,
      });
    }
  }, [namespace, namespaces, setValue]);

  const firstClassName = importClasses[0]?.metadata?.name;
  useEffect(() => {
    if (!backupClassName && firstClassName) {
      setValue(DatabaseImportFields.backupClassName, firstClassName, {
        shouldValidate: true,
      });
    }
  }, [backupClassName, firstClassName, setValue]);

  useEffect(() => {
    // A restored pick stays until the class's providers are known.
    if (isClassesLoading || isProvidersLoading) {
      return;
    }
    const names = classProviders.map((p) => p.metadata?.name ?? '');
    const nextName = names.includes(providerName)
      ? providerName
      : (names[0] ?? '');
    if (nextName !== providerName) {
      setValue(DatabaseImportFields.providerName, nextName, {
        shouldValidate: true,
      });
    }
  }, [
    classProviders,
    providerName,
    isClassesLoading,
    isProvidersLoading,
    setValue,
  ]);

  return {
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
  };
};
