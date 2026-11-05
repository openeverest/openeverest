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

import { useMonitoringBindingsForNamespaces } from 'hooks/api/monitoring-bindings';
import { useNamespacePermissionsForResource } from 'hooks/rbac';
import { groupInstancesByDestination } from '../../monitoring-destinations.utils';

// The Instances that send to the destination, or undefined when the user
// may not list them.
export const useDestinationInstances = (
  namespace: string,
  name: string
): string[] | undefined => {
  const { canRead } = useNamespacePermissionsForResource('monitoring-bindings');
  const readable = canRead.includes(namespace);
  const [bindings] = useMonitoringBindingsForNamespaces(
    readable ? [namespace] : []
  );

  if (!readable || !bindings?.data) {
    return undefined;
  }
  return groupInstancesByDestination(bindings.data).get(name) ?? [];
};
