import { UseMutationResult } from '@tanstack/react-query';
import { DeleteDbInstanceArgType } from 'hooks';
import { Instance } from 'types/api';

export interface DbActionsModalsProps {
  dbInstance: Instance;
  isNewClusterMode: boolean;
  openDetailsDialog?: boolean;
  handleCloseDetailsDialog?: () => void;
  openRestoreDialog: boolean;
  handleCloseRestoreDialog: () => void;
  openDeleteDialog: boolean;
  handleCloseDeleteDialog: () => void;
  handleConfirmDelete: (dataCheckbox: boolean) => void;
  deleteMutation: UseMutationResult<
    unknown,
    unknown,
    DeleteDbInstanceArgType,
    unknown
  >;
}
