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

import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';
import { IN_USE_NAMES_SHOWN } from '../monitoring-destinations.constants';
import { Messages } from '../monitoring-destinations.messages';
import { MonitoringDestinationDeletion } from '../monitoring-destinations.types';

interface DeleteDestinationDialogProps {
  deletion: MonitoringDestinationDeletion;
  isDeleting: boolean;
  onConfirm: () => void;
  onClose: () => void;
}

// A destination databases still use can't be deleted; the dialog names them.
export const DeleteDestinationDialog = ({
  deletion: { row, usedBy, secretDeletedToo },
  isDeleting,
  onConfirm,
  onClose,
}: DeleteDestinationDialogProps) =>
  usedBy.length > 0 ? (
    <ConfirmDialog
      open
      selectedId={row.name}
      closeModal={onClose}
      headerMessage={Messages.inUseHeader}
      cancelMessage={Messages.close}
      submitMessage=""
    >
      {Messages.inUse(row.name, usedBy, IN_USE_NAMES_SHOWN)}
    </ConfirmDialog>
  ) : (
    <ConfirmDialog
      open
      selectedId={row.name}
      selectedNamespace={row.namespace}
      closeModal={onClose}
      headerMessage={Messages.deleteHeader}
      cancelMessage={Messages.cancel}
      handleConfirmNamespace={onConfirm}
      disabledButtons={isDeleting}
    >
      {Messages.deleteConfirmation(row.name)}
      {secretDeletedToo && ` ${Messages.secretDeletedToo(secretDeletedToo)}`}
    </ConfirmDialog>
  );
