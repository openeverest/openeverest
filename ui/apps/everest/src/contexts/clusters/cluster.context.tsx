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

import { Alert, Box } from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import { getClustersFn } from 'api/clusters';
import LoadingPageSkeleton from 'components/loading-page-skeleton/LoadingPageSkeleton';
import AuthContext from 'contexts/auth/auth.context';
import { ReactNode, useContext, useEffect, useState } from 'react';
import { ClusterContext } from './cluster';

const SELECTED_CLUSTER_KEY = 'openeverest:selected-cluster';

export const ClusterProvider = ({ children }: { children: ReactNode }) => {
  const { authStatus } = useContext(AuthContext);
  const { data, error, isPending, refetch } = useQuery({
    queryKey: ['clusters'],
    queryFn: getClustersFn,
    enabled: authStatus === 'loggedIn',
  });
  const [selectedCluster, setSelectedCluster] = useState(
    () => localStorage.getItem(SELECTED_CLUSTER_KEY) ?? ''
  );

  useEffect(() => {
    if (!data?.items.length) {
      return;
    }
    if (!data.items.some((cluster) => cluster.name === selectedCluster)) {
      const fallback = data.items[0].name;
      setSelectedCluster(fallback);
      localStorage.setItem(SELECTED_CLUSTER_KEY, fallback);
    }
  }, [data, selectedCluster]);

  const selectCluster = (clusterName: string) => {
    setSelectedCluster(clusterName);
    localStorage.setItem(SELECTED_CLUSTER_KEY, clusterName);
  };

  const clusterName =
    authStatus === 'loggedIn'
      ? (data?.items.find((cluster) => cluster.name === selectedCluster)
          ?.name ??
        data?.items[0]?.name ??
        '')
      : '';

  if (authStatus === 'loggedIn' && isPending) {
    return <LoadingPageSkeleton />;
  }

  if (authStatus === 'loggedIn' && (error || !clusterName)) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">
          OpenEverest could not find a Kubernetes cluster available to manage.
        </Alert>
      </Box>
    );
  }

  return (
    <ClusterContext.Provider
      value={{
        clusterName,
        clusters: data?.items ?? [],
        selectCluster,
        refreshClusters: refetch,
      }}
    >
      {children}
    </ClusterContext.Provider>
  );
};
