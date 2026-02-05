import { MenuItem, Tooltip } from '@mui/material';
import { useRBACPermissions } from 'hooks/rbac';
import DeleteIcon from '@mui/icons-material/Delete';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import TableActionsMenu from 'components/table-actions-menu';

const SplitHorizonRowActions = ({
  namespace,
  configName,
  isConfigInUse,
  handleOnDeleteIconClick,
  handleOnEditIconClick,
}: {
  namespace: string;
  configName: string;
  isConfigInUse: boolean;
  handleOnDeleteIconClick: () => void;
  handleOnEditIconClick: () => void;
}) => {
  const { canDelete, canUpdate } = useRBACPermissions(
    'enginefeatures/split-horizon-dns-configs',
    `${namespace}/${configName}`
  );

  const menuItems = [
    ...(canUpdate
      ? [
          <Tooltip
            title={
              isConfigInUse
                ? 'This configuration is currently in use by one or more clusters. Please unassign it first before editing.'
                : undefined
            }
          >
            <span>
              <MenuItem
                key="edit"
                onClick={handleOnEditIconClick}
                disabled={isConfigInUse}
              >
                <EditOutlinedIcon sx={{ mr: 1 }} />
                Edit
              </MenuItem>
            </span>
          </Tooltip>,
        ]
      : []),
    ...(canDelete
      ? [
          <MenuItem key="delete" onClick={handleOnDeleteIconClick}>
            <DeleteIcon sx={{ mr: 1 }} />
            Delete
          </MenuItem>,
        ]
      : []),
  ];

  return (
    <TableActionsMenu
      menuItems={menuItems}
      isVertical
      buttonColor="primary.main"
    />
  );
};

export default SplitHorizonRowActions;
