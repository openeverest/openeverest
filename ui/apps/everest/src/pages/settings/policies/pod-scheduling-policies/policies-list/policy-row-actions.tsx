import { MenuItem } from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import Visibility from '@mui/icons-material/Visibility';
import TableActionsMenu from 'components/table-actions-menu';
import { useRBACPermissions } from 'hooks/rbac';
import { useNavigate } from 'react-router-dom';

type Props = {
  policyName: string;
  handleOnDeleteIconClick: () => void;
  readOnly: boolean;
};

const PolicyRowActions = ({
  policyName,
  handleOnDeleteIconClick,
  readOnly,
}: Props) => {
  const { canDelete } = useRBACPermissions(
    'pod-scheduling-policies',
    policyName
  );
  const navigate = useNavigate();

  return (
    <TableActionsMenu
      menuItems={[
        <MenuItem
          key="view"
          onClick={() =>
            navigate(`/settings/policies/details/pod-scheduling/${policyName}`)
          }
        >
          <Visibility sx={{ mr: 1 }} />
          View details
        </MenuItem>,
        canDelete && !readOnly ? (
          <MenuItem key="delete" onClick={handleOnDeleteIconClick}>
            <DeleteIcon sx={{ mr: 1 }} />
            Delete
          </MenuItem>
        ) : null,
      ]}
    />
  );
};

export default PolicyRowActions;
