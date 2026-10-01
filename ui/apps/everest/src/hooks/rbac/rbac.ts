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

import { useClusterName } from 'hooks/api/useClusterName';
import { useNamespaces } from 'hooks/api/namespaces';
import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  can,
  canAll,
  RBACAction,
  RBACStandardAction,
  AuthorizerObservable,
  RBACResource,
  toRBACObject,
} from 'utils/rbac';

export const useRBACPermissions = (
  resource: RBACResource,
  specificResources: string | string[] = '*'
) => {
  const [permissions, setPermissions] = useState<Record<RBACAction, boolean>>({
    read: false,
    update: false,
    create: false,
    delete: false,
    'read-connection': false,
  });
  const cluster = useClusterName();

  const checkPermissions = useCallback(async () => {
    const objects = (
      Array.isArray(specificResources) ? specificResources : [specificResources]
    ).map((o) => toRBACObject(resource, o, cluster));
    const check = (action: RBACAction) => canAll(action, resource, objects);
    const [canRead, canUpdate, canDelete, canCreate, canReadConnection] =
      await Promise.all([
        check('read'),
        check('update'),
        check('delete'),
        check('create'),
        check('read-connection'),
      ]);

    setPermissions({
      read: canRead,
      update: canUpdate,
      delete: canDelete,
      create: canCreate,
      'read-connection': canReadConnection,
    });
  }, [cluster, resource, specificResources]);

  useEffect(() => {
    checkPermissions();
    AuthorizerObservable.subscribe(checkPermissions);

    return () => AuthorizerObservable.unsubscribe(checkPermissions);
  }, [checkPermissions]);

  return {
    canRead: permissions.read,
    canReadConnection: permissions['read-connection'],
    canUpdate: permissions.update,
    canDelete: permissions.delete,
    canCreate: permissions.create,
  };
};

export const useNamespacePermissionsForResource = (
  resource: RBACResource,
  specificResource = '*'
) => {
  const [permissions, setPermissions] = useState<
    Record<RBACStandardAction, string[]>
  >({
    read: [],
    update: [],
    create: [],
    delete: [],
  });
  // Whether the async permission check for the current namespaces has resolved
  // at least once. Consumers gate UI on this to avoid flashing a pre-resolution
  // state (e.g. a "no permissions" empty state on remount when namespaces are
  // already cached and the check hasn't run yet).
  const [isPermissionsLoading, setIsPermissionsLoading] = useState(true);

  const cluster = useClusterName();
  const queryResult = useNamespaces();
  const { data: namespaces } = queryResult;

  const checkPermissions = useCallback(async () => {
    const newPermissions: Record<RBACStandardAction, string[]> = {
      read: [],
      update: [],
      create: [],
      delete: [],
    };
    const permissionsPromisesArr: Promise<void>[] = [];

    if (namespaces) {
      const actions: RBACStandardAction[] = [
        'read',
        'update',
        'delete',
        'create',
      ];
      for (const namespace of namespaces) {
        for (const action of actions) {
          permissionsPromisesArr.push(
            can(
              action,
              resource,
              toRBACObject(
                resource,
                `${namespace}/${specificResource}`,
                cluster
              )
            ).then((canDo) => {
              if (canDo) {
                newPermissions[action].push(namespace);
              }
            })
          );
        }
      }
    }
    await Promise.all(permissionsPromisesArr);
    setPermissions(newPermissions);
  }, [cluster, namespaces, resource, specificResource]);

  useEffect(() => {
    let active = true;
    setIsPermissionsLoading(true);
    AuthorizerObservable.subscribe(checkPermissions);
    checkPermissions().finally(() => {
      if (active) {
        setIsPermissionsLoading(false);
      }
    });
    return () => {
      active = false;
      AuthorizerObservable.unsubscribe(checkPermissions);
    };
  }, [checkPermissions]);

  return {
    canRead: permissions.read,
    canUpdate: permissions.update,
    canCreate: permissions.create,
    canDelete: permissions.delete,
    ...queryResult,
    isPermissionsLoading,
  };
};

export const useRBACPermissionRoute = (
  permissions: Array<{
    action: RBACAction;
    resource: RBACResource;
    specificResources?: string[];
  }>
) => {
  const navigate = useNavigate();
  const cluster = useClusterName();

  const checkPermissions = useCallback(async () => {
    for (let i = 0; i < permissions.length; i++) {
      const { action, resource, specificResources = [] } = permissions[i];
      const objects = specificResources.map((o) =>
        toRBACObject(resource, o, cluster)
      );
      const allowed = await canAll(action, resource, objects);
      if (!allowed) {
        navigate('/');
        return false;
      }
    }
  }, [cluster, navigate, permissions]);

  useEffect(() => {
    checkPermissions();
  }, [checkPermissions]);

  return true;
};
