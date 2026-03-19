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

import { lazy } from 'react';

export const Login = lazy(() =>
  import('pages/login').then((module) => ({ default: module.Login }))
);

export const DbClusterView = lazy(() =>
  import('pages/databases/DbClusterView').then((module) => ({
    default: module.DbClusterView,
  }))
);

export const DatabasePage = lazy(() =>
  import('pages/database-form/database-form').then((module) => ({
    default: module.DatabasePage,
  }))
);

export const DbClusterDetails = lazy(() =>
  import('pages/db-cluster-details/db-cluster-details').then((module) => ({
    default: module.DbClusterDetails,
  }))
);

export const ClusterOverview = lazy(() =>
  import('pages/db-cluster-details/cluster-overview/cluster-overview').then(
    (module) => ({
      default: module.ClusterOverview,
    })
  )
);

export const Settings = lazy(() =>
  import('pages/settings/settings').then((module) => ({
    default: module.Settings,
  }))
);

export const StorageLocations = lazy(() =>
  import('pages/settings/storage-locations/storage-locations').then(
    (module) => ({
      default: module.StorageLocations,
    })
  )
);

export const MonitoringEndpoints = lazy(() =>
  import('pages/settings/monitoring-endpoints/monitoring-endpoints').then(
    (module) => ({
      default: module.MonitoringEndpoints,
    })
  )
);

export const NoMatch = lazy(() =>
  import('pages/404/NoMatch').then((module) => ({
    default: module.NoMatch,
  }))
);

export const Backups = lazy(() =>
  import('pages/db-cluster-details/backups/backups').then((module) => ({
    default: module.Backups,
  }))
);

export const Namespaces = lazy(() =>
  import('pages/settings/namespaces/namespaces').then((module) => ({
    default: module.Namespaces,
  }))
);

export const NamespaceDetails = lazy(
  () => import('pages/settings/namespaces/namespace-details')
);

export const Restores = lazy(() => import('pages/db-cluster-details/restores'));

export const Components = lazy(
  () => import('pages/db-cluster-details/components')
);

export const Logs = lazy(
  () => import('pages/db-cluster-details/component-logs/component-logs')
);

export const LoginCallback = lazy(
  () => import('components/login-callback/LoginCallback')
);

export const Logout = lazy(() => import('pages/logout'));

export const Policies = lazy(() => import('pages/settings/policies/policies'));

export const PoliciesList = lazy(
  () => import('pages/settings/policies/pod-scheduling-policies/policies-list')
);

export const PolicyDetails = lazy(
  () => import('pages/settings/policies/pod-scheduling-policies/policy-details')
);

export const LoadBalancerConfiguration = lazy(
  () => import('pages/settings/policies/load-balancer-configuration')
);

export const LoadBalancerConfigDetails = lazy(
  () =>
    import(
      'pages/settings/policies/load-balancer-configuration/load-balancer-config-detials/load-balancer-config-detials'
    )
);

export const SettingsPoliciesRouter = lazy(
  () => import('pages/settings/settings-policies-router')
);

export const SplitHorizon = lazy(
  () => import('pages/settings/policies/split-horizon')
);
