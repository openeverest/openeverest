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

import {
  useNamespacePermissionsForResource,
  useRBACPermissions,
} from 'hooks/rbac';
import { MonitoringDestinationPermissions } from '../monitoring-destinations.types';
import { toWritableNamespaces } from '../monitoring-destinations.utils';

// The namespaces the user may add, edit and delete destinations in.
export const useMonitoringDestinationPermissions =
  (): MonitoringDestinationPermissions => {
    const destinationPermissions = useNamespacePermissionsForResource(
      'monitoring-destinations'
    );
    const secretPermissions = useNamespacePermissionsForResource('secrets');
    const { canRead: canReadClasses } =
      useRBACPermissions('monitoring-classes');

    return {
      createNamespaces: toWritableNamespaces({
        destinationNamespaces: destinationPermissions.canCreate,
        secretCreateNamespaces: secretPermissions.canCreate,
        secretReadNamespaces: secretPermissions.canRead,
        canReadClasses,
      }),
      updateNamespaces: toWritableNamespaces({
        destinationNamespaces: destinationPermissions.canUpdate,
        secretCreateNamespaces: secretPermissions.canCreate,
        secretReadNamespaces: secretPermissions.canRead,
        canReadClasses,
      }),
      deleteNamespaces: destinationPermissions.canDelete,
    };
  };
