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

import { useEffect, useMemo, useRef } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import { AuthenticationFormFields } from 'components/class-authentication';
import {
  BackupStorageClass,
  BackupStorageCRD,
} from 'shared-types/backupStorages.types';
import {
  getBackupStorageClassName,
  toBackupStorageClassContract,
} from 'utils/backup-storage-classes';
import { ClassContract } from 'utils/class-contract';
import { StorageFormFields } from '../backup-storage-modal.constants';
import {
  BackupStorageFormValues,
  StorageClassForms,
} from '../backup-storage-modal.types';
import {
  getFormMode,
  getNewStorageClassDefaults,
  getStorageClassForms,
} from '../backup-storage-modal.utils';

// The forms of the picked class and auth mode. Picking another class starts
// its forms from their defaults, and preselects a class's only auth mode; the
// credentials block drops what was picked or typed for the previous one.
export const useStorageClassForms = (
  classes: BackupStorageClass[],
  original: BackupStorageCRD | undefined
): {
  contract: ClassContract | undefined;
  forms: StorageClassForms;
} => {
  const { control, setValue } = useFormContext<BackupStorageFormValues>();
  const [className, authMode] = useWatch({
    control,
    name: [StorageFormFields.className, AuthenticationFormFields.authMode],
  });
  const storageClass = classes.find(
    (c) => getBackupStorageClassName(c) === className
  );
  const formMode = getFormMode(original);
  const forms = useMemo(
    () => getStorageClassForms(storageClass, authMode, formMode),
    [storageClass, authMode, formMode]
  );
  const contract = useMemo(
    () => storageClass && toBackupStorageClassContract(storageClass),
    [storageClass]
  );

  const previousClassName = useRef(className);
  useEffect(() => {
    if (previousClassName.current === className) {
      return;
    }
    previousClassName.current = className;
    const defaults = getNewStorageClassDefaults(storageClass);
    setValue(AuthenticationFormFields.authMode, defaults.authMode, {
      shouldValidate: true,
    });
    setValue(StorageFormFields.spec, defaults.spec);
    setValue(StorageFormFields.fallbackParameters, defaults.fallbackParameters);
  }, [className, storageClass, setValue]);

  return { contract, forms };
};
