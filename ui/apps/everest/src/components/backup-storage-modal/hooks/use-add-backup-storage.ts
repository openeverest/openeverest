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

import { useState } from 'react';
import { BackupStorageModalProps } from '../backup-storage-modal.types';
import { useBackupStoragePermissions } from './use-backup-storage-permissions';

// Adds a storage to a namespace outside Settings, where a form needs one,
// through the same dialog.
export const useAddBackupStorage = (namespace: string) => {
  const [isOpen, setIsOpen] = useState(false);
  const { createNamespaces } = useBackupStoragePermissions();
  const modalProps: BackupStorageModalProps = {
    namespace,
    onClose: () => setIsOpen(false),
  };

  return {
    canAdd: createNamespaces.includes(namespace),
    isOpen,
    open: () => setIsOpen(true),
    modalProps,
  };
};
