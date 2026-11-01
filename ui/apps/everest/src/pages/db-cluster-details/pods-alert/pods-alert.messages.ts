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

import type { PodsAlertReason } from 'shared-types/instance.types';

export const Messages: {
  reasonLabel: string;
  alerts: Record<PodsAlertReason, { title: string; hint: string }>;
} = {
  reasonLabel: 'Reason:',
  alerts: {
    Unschedulable: {
      title: 'Some pods cannot be scheduled',
      hint: "No node meets these pods' requirements. Add nodes, free up CPU or memory, or change the scheduling policy.",
    },
    CrashLoopBackOff: {
      title: 'Some pods keep crashing',
      hint: 'A container exits soon after it starts and keeps restarting. Check the pod logs, or raise the memory limit if it ran out of memory.',
    },
    ImagePullBackOff: {
      title: 'Some pods cannot pull their image',
      hint: 'The cluster cannot download a container image. Check that the image exists and that the cluster can reach its registry.',
    },
    CreateContainerConfigError: {
      title: 'Some pods cannot start',
      hint: 'A Secret or ConfigMap these pods use is missing or invalid. Restore it and the pods start on their own.',
    },
  },
};
