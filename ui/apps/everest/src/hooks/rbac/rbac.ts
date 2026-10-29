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

// Runs the check now and on every policy change. Permissions start denied, so
// the flag tells "denied" from "not checked yet".
const useAuthorizerCheck = (check: () => Promise<void>) => {
  const [isPermissionsLoading, setIsPermissionsLoading] = useState(true);

  useEffect(() => {
    let active = true;
    setIsPermissionsLoading(true);
    AuthorizerObservable.subscribe(check);
    check().finally(() => {
      if (active) {
        setIsPermissionsLoading(false);
      }
    });
    return () => {
      active = false;
      AuthorizerObservable.unsubscribe(check);
    };
  }, [check]);

  return isPermissionsLoading;
};

export const useRBACPermissions = (
  resource: RBACResource,
  specificResources: string | string[] = '*'
) => {
  const [permissions, setPermissions] = useState<
    Record<RBACStandardAction, boolean>
  >({
    read: false,
    update: false,
    create: false,
    delete: false,
  });
  const cluster = useClusterName();

  const checkPermissions = useCallback(async () => {
    const objects = (
      Array.isArray(specificResources) ? specificResources : [specificResources]
    ).map((o) => toRBACObject(resource, o, cluster));
    const check = (action: RBACStandardAction) =>
      canAll(action, resource, objects);
    const [canRead, canUpdate, canDelete, canCreate] = await Promise.all([
      check('read'),
      check('update'),
      check('delete'),
      check('create'),
    ]);

    setPermissions({
      read: canRead,
      update: canUpdate,
      delete: canDelete,
      create: canCreate,
    });
  }, [cluster, resource, specificResources]);

  const isPermissionsLoading = useAuthorizerCheck(checkPermissions);

  return {
    canRead: permissions.read,
    canUpdate: permissions.update,
    canDelete: permissions.delete,
    canCreate: permissions.create,
    isPermissionsLoading,
  };
};

// Actions beyond CRUD belong to one resource (e.g. instances:read-connection),
// so they are checked on demand instead of in every useRBACPermissions call.
export const useRBACAction = (
  action: RBACAction,
  resource: RBACResource,
  specificResource: string
) => {
  const [isAllowed, setIsAllowed] = useState(false);
  const cluster = useClusterName();

  const checkPermission = useCallback(async () => {
    setIsAllowed(
      await can(
        action,
        resource,
        toRBACObject(resource, specificResource, cluster)
      )
    );
  }, [action, cluster, resource, specificResource]);

  const isPermissionsLoading = useAuthorizerCheck(checkPermission);

  return { isAllowed, isPermissionsLoading };
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

  // Consumers gate UI on this to avoid flashing a "no permissions" state when
  // namespaces are already cached and the check hasn't run yet.
  const isPermissionsLoading = useAuthorizerCheck(checkPermissions);

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
