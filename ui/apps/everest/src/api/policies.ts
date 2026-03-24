import { api } from './api';
import rbac from '/src/static/model.conf?raw';

type RBACPoliciesPayload = {
  permissions: string[][];
  enabled?: boolean;
};

export type RBACPolicies = {
  enabled: boolean;
  m: string;
  p: string[][];
};

export const getRBACPolicies = async (): Promise<RBACPolicies> => {
  const response = await api.get<RBACPoliciesPayload>('permissions');
  const permissions = (response.data.permissions || []).map(
    (permission: string[]) => {
      return [...'p', ...permission];
    }
  );

  return {
    enabled: response.data.enabled || false,
    m: rbac,
    p: permissions || [],
  };
};
