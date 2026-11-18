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

// The label a person reads for each status word. The vocabulary is open: a
// reason missing here is split into words by getReasonLabel, so a new one
// needs no change in this file to be shown. Kept identical to the dictionary
// everestctl uses: a test compares it with the fixtures both share.
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
  NotResponding: 'Provider not responding',
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
};

export const Messages = {
  // Whether clients can connect, said next to the status word while work is
  // in flight: "Upgrading · online".
  connectivity: {
    Yes: 'online',
    No: 'offline',
  },
  withDetail: (statusWord: string, detail: string) =>
    `${statusWord} \u00b7 ${detail}`,
};
