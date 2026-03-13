import { Navigate, createBrowserRouter } from 'react-router-dom';
import ProtectedRoute from 'components/protected-route/ProtectedRoute';
import { Main } from 'components/main/Main';
import { DBClusterDetailsTabs } from 'pages/db-cluster-details/db-cluster-details.types';
import { SettingsTabs } from 'pages/settings/settings.types';
import { DbClusterContextProvider } from 'pages/db-cluster-details/dbCluster.context';
import {
  Backups,
  ClusterOverview,
  Components,
  DatabasePage,
  DbClusterDetails,
  DbClusterView,
  LoadBalancerConfigDetails,
  LoadBalancerConfiguration,
  Login,
  LoginCallback,
  Logout,
  Logs,
  MonitoringEndpoints,
  NamespaceDetails,
  Namespaces,
  NoMatch,
  Policies,
  PoliciesList,
  PolicyDetails,
  Restores,
  Settings,
  SettingsPoliciesRouter,
  SplitHorizon,
  StorageLocations,
} from './router-lazy-pages';
import { withSuspense } from './router-suspense';

const router = createBrowserRouter([
  {
    path: 'login',
    element: withSuspense(<Login />),
  },
  {
    path: '/login-callback',
    element: withSuspense(<LoginCallback />),
  },
  {
    path: '/logout',
    element: withSuspense(<Logout />),
  },
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <Main />
      </ProtectedRoute>
    ),
    children: [
      {
        path: 'databases',
        element: withSuspense(<DbClusterView />),
      },
      {
        path: 'databases/new',
        element: withSuspense(<DatabasePage />),
      },
      {
        path: 'databases/:namespace/:dbClusterName',
        element: withSuspense(
          <DbClusterContextProvider>
            <DbClusterDetails />
          </DbClusterContextProvider>
        ),
        children: [
          {
            path: DBClusterDetailsTabs.backups,
            element: withSuspense(<Backups />),
          },
          {
            index: true,
            path: DBClusterDetailsTabs.overview,
            element: withSuspense(<ClusterOverview />),
          },
          {
            path: DBClusterDetailsTabs.components,
            element: withSuspense(<Components />),
          },
          {
            path: DBClusterDetailsTabs.restores,
            element: withSuspense(<Restores />),
          },
          {
            path: DBClusterDetailsTabs.logs,
            element: withSuspense(<Logs />),
          },
        ],
      },
      {
        index: true,
        element: <Navigate to="/databases" replace />,
      },
      {
        path: 'settings',
        element: withSuspense(<Settings />),
        children: [
          {
            path: SettingsTabs.storageLocations,
            element: withSuspense(<StorageLocations />),
          },
          {
            path: SettingsTabs.monitoringEndpoints,
            element: withSuspense(<MonitoringEndpoints />),
          },
          {
            path: SettingsTabs.namespaces,
            element: withSuspense(<Namespaces />),
          },
          {
            path: SettingsTabs.policies,
            element: withSuspense(<Policies />),
          },
        ],
      },
      {
        path: '/settings/policies/details',
        element: withSuspense(<SettingsPoliciesRouter />),
        children: [
          {
            path: 'pod-scheduling',
            element: withSuspense(<PoliciesList />),
          },
          {
            path: 'pod-scheduling/:name',
            element: withSuspense(<PolicyDetails />),
          },
          {
            path: 'load-balancer-configuration',
            element: withSuspense(<LoadBalancerConfiguration />),
          },
          {
            path: 'load-balancer-configuration/:configName',
            element: withSuspense(<LoadBalancerConfigDetails />),
          },
          {
            path: 'split-horizon',
            element: withSuspense(<SplitHorizon />),
          },
          {
            index: true,
            element: <Navigate to="pod-scheduling" replace />,
          },
        ],
      },
      {
        path: '/settings/namespaces/:namespace',
        element: withSuspense(<NamespaceDetails />),
      },
      {
        path: '*',
        element: withSuspense(<NoMatch />),
      },
    ],
  },
]);

export default router;
