import { Visibility } from '@mui/icons-material';
import { MenuItem } from '@mui/material';
import TableActionsMenu from 'components/table-actions-menu';
import { useNavigate } from 'react-router-dom';
import DeleteIcon from '@mui/icons-material/Delete';
import { messages } from '../load-balancer.messages';
import { useRBACPermissions } from 'hooks/rbac/rbac';

interface LoadBalancerRowActionsProps {
  configName: string;
  readOnly?: boolean;
  handleOnDeleteIconClick: () => void;
}

const LoadBalancerRowActions = ({
  configName,
  handleOnDeleteIconClick,
  readOnly,
}: LoadBalancerRowActionsProps) => {
  const { canDelete } = useRBACPermissions(
    'load-balancer-configs',
    `${configName}`
  );
  const navigate = useNavigate();
  const menuItems = readOnly
    ? [
        <MenuItem
          key="view"
          onClick={() =>
            navigate(
              `/settings/policies/details/load-balancer-configuration/${configName}`
            )
          }
        >
          <Visibility sx={{ mr: 1 }} />
          {messages.rowActions.viewDetails}
        </MenuItem>,
      ]
    : [
        <MenuItem
          key="view"
          onClick={() =>
            navigate(
              `/settings/policies/details/load-balancer-configuration/${configName}`
            )
          }
        >
          <Visibility sx={{ mr: 1 }} />
          {messages.rowActions.viewDetails}
        </MenuItem>,
        canDelete && (
          <MenuItem key="delete" onClick={handleOnDeleteIconClick}>
            <DeleteIcon sx={{ mr: 1 }} />
            {messages.rowActions.delete}
          </MenuItem>
        ),
      ];
  return (
    <TableActionsMenu
      menuItems={menuItems}
      isVertical
      buttonColor="primary.main"
    />
  );
};

export default LoadBalancerRowActions;
