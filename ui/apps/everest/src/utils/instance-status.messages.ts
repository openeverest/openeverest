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
  InstanceAvailability,
  InstanceTone,
} from 'shared-types/instance.types';

// The label a person reads for each reason the Ready condition is known to
// carry. The vocabulary is open: a reason missing here is split into words by
// getReasonLabel, so a new one never needs a change in this file to be shown.
// Kept identical to the dictionary everestctl uses; a shared test compares
// both against the same fixtures.
export const REASON_LABELS: Record<string, string> = {
  Ready: 'Ready',
  Suspended: 'Stopped',
  Creating: 'Creating',
  ApplyingChanges: 'Applying changes',
  Deleting: 'Deleting',
  Restoring: 'Restoring',
  Upgrading: 'Upgrading',
  StorageResizing: 'Resizing storage',
  Scaling: 'Scaling',
  ValidationFailed: 'Invalid configuration',
  AwaitingApproval: 'Awaiting approval',
  RetriesExhausted: 'Maintenance failed',
  ProviderNotFound: 'Provider not found',
  UserSecretNotFound: 'Secret not found',
  InvalidReference: 'Invalid reference',
  Unschedulable: "Pods can't be scheduled",
  CrashLoopBackOff: 'Crash loop',
  ImagePullBackOff: "Image can't be pulled",
  CreateContainerConfigError: "Container can't be created",
  NotReady: 'Pods starting',
  ReconcileError: 'Provider error',
  EngineError: 'Engine error',
  ProgressDeadlineExceeded: 'Not progressing',
  BackupStorageNotFound: 'Backup storage not found',
  BackupClassNotFound: 'Backup class not found',
  BackupStorageCredentialsNotFound: 'Backup credentials not found',
  NotPickedUp: 'Not picked up by provider',
};

export const Messages = {
  up: 'Up',
  down: 'Down',
  deleting: 'Deleting',
  sentenceSeparator: ' \u00b7 ',
  // Shown where the status has no message, because it says nothing about
  // the spec that is stored.
  notObserved: 'The provider has not picked up the latest configuration yet.',
  notPickedUp:
    'No provider has picked this up for more than 5 minutes. Check that the provider is installed and running.',
  beingDeleted: 'The database is being deleted.',
  availability: {
    Yes: 'Yes',
    No: 'No',
    Unknown: '\u2014',
  } satisfies Record<InstanceAvailability, string>,
  availabilityHint: {
    Yes: 'Clients can connect.',
    No: 'Clients cannot connect.',
    Unknown: 'Whether clients can connect is not known.',
  } satisfies Record<InstanceAvailability, string>,
  tones: {
    error: 'Error',
    attention: 'Warning',
    unknown: 'Unknown',
    progress: 'In progress',
    stopped: 'Stopped',
    deleting: 'Deleting',
    ok: 'OK',
  } satisfies Record<InstanceTone, string>,
};
