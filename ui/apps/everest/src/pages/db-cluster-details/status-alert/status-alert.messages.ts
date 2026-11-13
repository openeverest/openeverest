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

export const Messages = {
  details: 'Details:',
  observed: (elapsed: string) => `First observed ${elapsed} ago.`,
  // Whether waiting can still help.
  retrying:
    'OpenEverest keeps retrying, so this may clear by itself. If it lasts, the database is reported as stalled.',
  stalled: 'This will not clear by itself: someone has to fix the cause.',
};

// What to do about the reasons the UI has advice for. A reason that is not
// here, such as one a provider defines, shows what the provider reported and
// nothing else.
export const REASON_HINTS: Record<string, string> = {
  Unschedulable:
    "No node meets these pods' requirements. Add nodes, free up CPU or memory, or change the scheduling policy.",
  CrashLoopBackOff:
    'A container exits soon after it starts and keeps restarting. Check the pod logs, or raise the memory limit if it ran out of memory.',
  ImagePullBackOff:
    'The cluster cannot download a container image. Check that the image exists and that the cluster can reach its registry.',
  CreateContainerConfigError:
    'A Secret or ConfigMap these pods use is missing or invalid. Restore it and the pods start on their own.',
  ValidationFailed:
    'The provider cannot apply this configuration. Edit the database to correct it.',
  ProviderNotFound:
    'The provider this database belongs to is not installed. Install it and the database is picked up on its own.',
  UserSecretNotFound:
    'The Secret holding the database users does not exist. Create it and the database carries on.',
  InvalidReference:
    'An object this database refers to exists but does not fit it. Correct that object or refer to another one.',
  BackupStorageNotFound:
    "A backup storage this database uses does not exist. Create it again, or remove it from the database's backups.",
  BackupStorageCredentialsNotFound:
    'The credentials Secret of a backup storage this database uses does not exist. Create it again.',
  BackupClassNotFound: 'The backup class this database uses is not installed.',
};
