// everest
// Copyright (C) 2023 Percona LLC
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
export type KubernetesCluster = {
  id: string;
  name: string;
};

export type KubernetesClusterList = KubernetesCluster[];

export type StorageClass = {
  allowVolumeExpansion: boolean;
  metadata: {
    name: string;
    annotations?: Record<string, string>;
  };
};

export type GetKubernetesClusterInfoPayload = {
  clusterType: string;
  storageClassNames: string[];
  storageClasses?: StorageClass[];
};

export type KubernetesClusterInfo = GetKubernetesClusterInfoPayload;

type Resource = {
  cpuMillis: number;
  memoryBytes: number;
  diskSize: number;
};
export type GetKubernetesClusterResourcesInfoPayload = {
  capacity: Resource;
  available: Resource;
};

export type KubernetesClusterResourcesInfo =
  GetKubernetesClusterResourcesInfoPayload;

export type GetKubernetesClusterMonitoringInfoResponse = {
  clusterType: string;
  storageClassNames: string[];
};

export type KubernetesClusterMonitoringInfo =
  GetKubernetesClusterMonitoringInfoResponse;
