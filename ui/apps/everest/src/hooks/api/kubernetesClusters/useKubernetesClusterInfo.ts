import { QueryKey, useQuery } from '@tanstack/react-query';
import {
  GetKubernetesClusterInfoPayload,
  KubernetesClusterInfo,
} from 'shared-types/kubernetes.types';
import { getKubernetesClusterInfoFn } from 'api/kubernetesClusterApi';

export const KUBERNETES_CLUSTER_INFO_QUERY_KEY = ['kubernetesClusterInfo'];

export const useKubernetesClusterInfo = (queryKey: QueryKey) =>
  useQuery<GetKubernetesClusterInfoPayload, unknown, KubernetesClusterInfo>({
    queryKey,
    queryFn: getKubernetesClusterInfoFn,
  });
