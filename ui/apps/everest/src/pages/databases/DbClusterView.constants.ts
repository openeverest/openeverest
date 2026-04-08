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

import { BaseStatus } from 'components/status-field/status-field.types';
import { DbInstancePhase } from 'shared-types/instance.types';

export const DB_INSTANCE_STATUS_TO_BASE_STATUS: Record<
  DbInstancePhase,
  BaseStatus
> = {
  Failed: 'error',
  Initializing: 'pending',
  Pending: 'pending',
  Provisioning: 'pending',
  Ready: 'success',
  Restoring: 'pending',
  Resuming: 'pending',
  Suspended: 'paused',
  Suspending: 'pending',
  Terminating: 'deleting',
  Updating: 'pending',
  Unknown: 'unknown',
  //   [DbClusterStatus.stopping]: 'pending',
  //   [DbClusterStatus.resizingVolumes]: 'pending',
  //   [DbClusterStatus.upgrading]: 'upgrading',
  //   [DbClusterStatus.importing]: 'importing',
};
