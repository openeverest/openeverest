import { Alert, Box, Skeleton, Tab, Tabs } from '@mui/material';
import {
  Link,
  Navigate,
  Outlet,
  useMatch,
  useNavigate,
  useParams,
} from 'react-router-dom';
import { NoMatch } from '../404/NoMatch';
import BackNavigationText from 'components/back-navigation-text';
import { DBClusterDetailsTabs } from './db-cluster-details.types';
import { DbClusterStatus } from 'shared-types/dbCluster.types';
import { DbClusterContext } from './dbCluster.context';
import { useContext } from 'react';
import { DB_CLUSTER_STATUS_TO_BASE_STATUS } from '../databases/DbClusterView.constants';
import { beautifyDbClusterStatus } from '../databases/DbClusterView.utils';
import StatusField from 'components/status-field';
import DbActions from 'components/db-actions/db-actions';
import { Messages } from './db-cluster-details.messages';
import { useRBACPermissionRoute } from 'hooks/rbac';
import DeletedDbDialog from './deleted-db-dialog';

const WithPermissionDetails = ({
  namespace,
  dbClusterName,
  tab,
}: {
  namespace: string;
  dbClusterName: string;
  tab: string;
}) => {
  const { dbCluster, clusterDeleted } = useContext(DbClusterContext);
  const navigate = useNavigate();

  useRBACPermissionRoute([
    {
      action: 'read',
      resource: 'database-clusters',
      specificResources: [`${namespace}/${dbClusterName}`],
    },
  ]);

  const tabs = Object.keys(DBClusterDetailsTabs) as Array<
    keyof typeof DBClusterDetailsTabs
  >;

  return (
    <>
      <Box sx={{ width: '100%' }}>
        <Box
          sx={{
            display: 'flex',
            gap: 1.5,
            alignItems: 'center',
            justifyContent: 'flex-start',
            mb: 1,
          }}
        >
          <BackNavigationText
            text={dbClusterName!}
            onBackClick={() => navigate('/databases')}
          />
          {/* At this point, loading is done and we either have the cluster or not */}
          <Box
            sx={{
              display: 'flex',
              justifyContent: 'space-between',
              flex: '1 0 auto',
              alignItems: 'center',
            }}
          >
            <StatusField
              dataTestId={dbClusterName}
              status={dbCluster?.status?.status || DbClusterStatus.creating}
              statusMap={DB_CLUSTER_STATUS_TO_BASE_STATUS}
            >
              {beautifyDbClusterStatus(
                dbCluster?.status?.status || DbClusterStatus.creating,
                dbCluster?.status?.conditions || []
              )}
            </StatusField>
            <DbActions showStatusActions={true} dbCluster={dbCluster!} />
          </Box>
        </Box>
        <Box
          sx={{
            borderBottom: 1,
            borderColor: 'divider',
            mb: 1,
          }}
        >
          <Tabs
            value={tab}
            variant="scrollable"
            allowScrollButtonsMobile
            aria-label="nav tabs"
          >
            {tabs.map((item) => (
              <Tab
                // @ts-ignore
                label={Messages[item]}
                // @ts-ignore
                key={DBClusterDetailsTabs[item]}
                // @ts-ignore
                value={DBClusterDetailsTabs[item]}
                // @ts-ignore
                to={DBClusterDetailsTabs[item]}
                component={Link}
                data-testid={`${
                  DBClusterDetailsTabs[item as DBClusterDetailsTabs]
                }`}
              />
            ))}
          </Tabs>
        </Box>
        {dbCluster!.status?.status === DbClusterStatus.restoring && (
          <Alert severity="warning" sx={{ my: 1 }}>
            {Messages.restoringDb}
          </Alert>
        )}
        <Outlet />
      </Box>
      {clusterDeleted && <DeletedDbDialog dbClusterName={dbClusterName} />}
    </>
  );
};

export const DbClusterDetails = () => {
  const { dbClusterName = '' } = useParams();

  const { dbCluster, isLoading } = useContext(DbClusterContext);
  const routeMatch = useMatch('/databases/:namespace/:dbClusterName/:tabs');
  const currentTab = routeMatch?.params?.tabs;
  const namespace = routeMatch?.params?.namespace;

  if (!currentTab) {
    return <Navigate to={DBClusterDetailsTabs.overview} replace />;
  }

  if (isLoading) {
    return (
      <>
        <Skeleton variant="rectangular" />
        <Skeleton variant="rectangular" />
        <Skeleton />
        <Skeleton />
        <Skeleton />
        <Skeleton variant="rectangular" />
      </>
    );
  }

  // We went through the array and know the cluster is not there. Safe to show 404
  if (!dbCluster) {
    return <NoMatch />;
  }

  // All clear, show the cluster data
  return (
    <WithPermissionDetails
      namespace={namespace!}
      dbClusterName={dbClusterName}
      tab={currentTab}
    />
  );
};
