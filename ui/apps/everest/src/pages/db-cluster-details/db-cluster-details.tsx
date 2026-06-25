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

import PendingIcon from '@mui/icons-material/Pending';
import { Box, Skeleton, Tab, Tabs } from '@mui/material';
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
import { DbInstanceContext } from './dbCluster.context';
import { useContext, useMemo } from 'react';
import DbActions from 'components/db-actions/db-actions';
import { Messages } from './db-cluster-details.messages';
import StatusField from 'components/status-field';
import { DB_INSTANCE_STATUS_TO_BASE_STATUS } from 'pages/databases/DbClusterView.constants';
import { beautifyDbInstanceStatus } from 'pages/databases/DbClusterView.utils';
import {
  DB_INSTANCE_UNKNOWN_PHASE,
  DbInstancePhase,
} from 'shared-types/instance.types';
import { usePlugins } from 'contexts/plugins';
import type { ClusterDetailTabExtension } from '@openeverest/plugin-sdk';

const WithPermissionDetails = ({
  instanceName,
  tab,
}: {
  instanceName: string;
  tab: string;
}) => {
  const { instance /*clusterDeleted */ } = useContext(DbInstanceContext);
  const navigate = useNavigate();
  const status =
    (instance?.status?.phase as DbInstancePhase | undefined) ??
    DB_INSTANCE_UNKNOWN_PHASE;

  // TODO RBAC
  // useRBACPermissionRoute([
  //   {
  //     action: 'read',
  //     resource: 'database-clusters',
  //     specificResources: [`${namespace}/${instanceName}`],
  //   },
  // ]);

  const tabs = Object.keys(DBClusterDetailsTabs) as Array<
    keyof typeof DBClusterDetailsTabs
  >;

  // Collect clusterDetailTab extensions, filtered by engine type
  // (providers field). Per-namespace plugin visibility is governed by
  // Everest RBAC on the `/v1/plugins` endpoint.
  const { plugins } = usePlugins();
  const engineType = instance?.spec?.provider;

  const pluginTabs = useMemo(
    () =>
      plugins.flatMap((p) =>
        p.extensions
          .filter(
            (ext): ext is ClusterDetailTabExtension =>
              ext.type === 'clusterDetailTab'
          )
          .filter(
            (ext) =>
              !ext.providers?.length ||
              (engineType != null && ext.providers.includes(engineType))
          )
          .map((ext) => ({ pluginName: p.name, ...ext }))
      ),
    [plugins, engineType]
  );

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
            text={instanceName!}
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
              dataTestId={instanceName}
              status={status}
              statusMap={DB_INSTANCE_STATUS_TO_BASE_STATUS}
              defaultIcon={PendingIcon}
            >
              {beautifyDbInstanceStatus(status)}
            </StatusField>
            <DbActions showStatusActions dbInstance={instance!} />
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
            aria-label="instance detail tabs"
          >
            {tabs.map((item) => (
              <Tab
                label={Messages[item]}
                key={DBClusterDetailsTabs[item]}
                value={DBClusterDetailsTabs[item]}
                to={DBClusterDetailsTabs[item]}
                component={Link}
                data-testid={`${DBClusterDetailsTabs[item]}`}
              />
            ))}
            {pluginTabs.map((pt) => (
              <Tab
                label={pt.label}
                key={`plugin-${pt.pluginName}-${pt.path}`}
                value={pt.path}
                to={pt.path}
                component={Link}
                data-testid={`plugin-tab-${pt.path}`}
              />
            ))}
          </Tabs>
        </Box>
        {/*TODO return when statuses will be ready */}
        {/* {instance!.status?.status === DbInstanceStatus.restoring && (
          <Alert severity="warning" sx={{ my: 1 }}>
            {Messages.restoringDb}
          </Alert>
        )} */}
        <Outlet />
      </Box>
    </>
  );
};

export const DbClusterDetails = () => {
  const { instanceName = '' } = useParams();
  const { instance, isLoading } = useContext(DbInstanceContext);
  const routeMatch = useMatch('/databases/:namespace/:instanceName/:tabs');
  const currentTab = routeMatch?.params?.tabs;

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

  if (!instance) {
    return <NoMatch />;
  }

  // All clear, show the cluster data
  return <WithPermissionDetails instanceName={instanceName} tab={currentTab} />;
};
