import { Box, Typography } from '@mui/material';
import BackTo from './policies/shared/back-to';
import { Outlet, useMatch } from 'react-router-dom';

const messages = {
  routeMessages: {
    'pod-scheduling': 'Pod Scheduling Policies',
    'load-balancer-configuration': 'Load Balancer Configuration',
    'split-horizon': 'Split-Horizon DNS',
  },
};

const SettingsPoliciesRouter = () => {
  const routeMatch = useMatch('/settings/policies/details/:routeName');
  const subRouteMatch = useMatch(
    '/settings/policies/details/:routeName/:subRouteName'
  );
  const currentRoute = routeMatch?.params?.routeName;
  return (
    <Box>
      {!subRouteMatch && (
        <BackTo to="/settings/policies" prevPage="all policies" />
      )}
      <Typography variant="h6" sx={{ mt: 2, mb: 1 }}>
        {
          messages.routeMessages[
            currentRoute as keyof typeof messages.routeMessages
          ]
        }
      </Typography>
      <Outlet />
    </Box>
  );
};

export default SettingsPoliciesRouter;
