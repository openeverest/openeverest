import { CustomConfirmDialog } from 'components/custom-confirm-dialog';

type Props = {
  configName: string;
  submitting?: boolean;
  configInUse?: boolean;
  handleConfirmDelete?: () => void;
  handleCloseDeleteDialog?: () => void;
};
const DeleteSplitHorizonConfigDialog = ({
  configName,
  submitting,
  configInUse = false,
  handleConfirmDelete = () => {},
  handleCloseDeleteDialog = () => {},
}: Props) => (
  <CustomConfirmDialog
    alertTitle={configInUse ? 'Configuration in use' : undefined}
    confirmationInput={!configInUse}
    disableSubmit={configInUse}
    inputLabel="Config name"
    inputPlaceholder="Config name"
    selectedId={configName}
    isOpen
    closeModal={handleCloseDeleteDialog}
    headerMessage="Delete config"
    submitting={submitting}
    handleConfirm={handleConfirmDelete}
    alertMessage={
      configInUse
        ? 'This configuration is currently in use by one or more clusters. Please unassign it first before deleting.'
        : 'This action will permanently delete your configuration and affect all the clusters using it.'
    }
    dialogContent={
      !configInUse && (
        <>
          Are you sure you want to permanently delete <b>{configName}</b>{' '}
          policy? To confirm this action, type the name of your policy.
        </>
      )
    }
    hideCheckbox
  />
);

export default DeleteSplitHorizonConfigDialog;
