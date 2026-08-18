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

import { Instance } from 'shared-types/api.types';
import { PitrWindow } from 'shared-types/pitr.types';
import { resolvePitrWindow } from 'utils/pitr';
import { InstanceBackupStorage } from './backups.types';
import { PitrParameters } from 'components/pitr-config-modal/pitr-config-modal.types';

// Look up a storage's PITR status on the instance and resolve its window;
// undefined when the storage reports no PITR status at all.
export const getStoragePitrWindow = (
  instance: Instance,
  storageName: string
): PitrWindow | undefined => {
  const status = instance.status?.backup?.storages?.find(
    (storage) => storage.name === storageName
  )?.pitr;
  return status ? resolvePitrWindow(status) : undefined;
};

// Merge a PITR patch onto the existing storage pitr. Generated CRD types model
// pitr.parameters as Record<string, never>; the payload is provider-defined, so
// the single unavoidable widening to the CRD type is centralized here at the
// write boundary, keeping call sites free of casts.
export const buildPitrPayload = (
  previous: { enabled?: boolean; parameters?: PitrParameters } | undefined,
  patch: { enabled: boolean; parameters?: PitrParameters }
): InstanceBackupStorage['pitr'] =>
  ({ ...previous, ...patch }) as unknown as InstanceBackupStorage['pitr'];

export const setStoragePitr = (
  storages: InstanceBackupStorage[],
  storageName: string,
  pitr: InstanceBackupStorage['pitr']
): InstanceBackupStorage[] =>
  storages.map((storage) =>
    storage.storageRef.name === storageName ? { ...storage, pitr } : storage
  );

export const countPitrEnabledStorages = (
  storages: InstanceBackupStorage[]
): number => storages.filter((storage) => storage.pitr?.enabled).length;

export type PitrBlockReason = 'limit-reached';

// Shared gating for turning PITR on, used by both the cluster-details storages
// panel and the creation wizard. Works on primitives (not a storage type) so
// each surface computes the inputs from its own shape. Returns a reason code
// (not a message) so callers own their copy; undefined means enabling is
// allowed. The only hard block is the per-provider enabled-storages limit — a
// missing backup/schedule is a non-blocking warning (see
// shouldWarnPitrNeedsBackup), and an already-enabled storage is never blocked.
export const getPitrBlockReason = ({
  enabledCount,
  maxEnabled,
  storageEnabled,
}: {
  enabledCount: number;
  maxEnabled: number | undefined;
  storageEnabled: boolean;
}): PitrBlockReason | undefined => {
  if (storageEnabled) {
    return undefined;
  }
  if (maxEnabled != null && enabledCount >= maxEnabled) {
    return 'limit-reached';
  }
  return undefined;
};
