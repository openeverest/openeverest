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
  InstanceConditionType,
  InstanceProblem,
  InstanceStatusDisplay,
  InstanceStatusSource,
  InstanceTone,
} from 'shared-types/instance.types';
import { Messages, REASON_LABELS } from './instance-status.messages';

const REASON_CREATING = 'Creating';
const REASON_APPLYING_CHANGES = 'ApplyingChanges';
const REASON_DELETING = 'Deleting';

const PHASE_PENDING = 'Pending';
const PHASE_PROVISIONED = 'Provisioned';

export const getInstanceCondition = (
  instance: InstanceStatusSource | undefined,
  type: string
) => instance?.status?.conditions?.find((condition) => condition.type === type);

export const isInstanceConditionTrue = (
  instance: InstanceStatusSource | undefined,
  type: string
) => getInstanceCondition(instance, type)?.status === 'True';

const isUpper = (char: string) => /\p{Lu}/u.test(char);
const isLower = (char: string) => /\p{Ll}/u.test(char);
const isLetterOrDigit = (char: string) => /[\p{L}\p{Nd}]/u.test(char);

// Whether the capital at index begins a word: it follows a lowercase letter
// or a digit, or it ends a run of capitals.
const startsWord = (chars: string[], index: number) => {
  if (!isUpper(chars[index])) {
    return false;
  }
  if (!isUpper(chars[index - 1])) {
    return isLetterOrDigit(chars[index - 1]);
  }
  return index + 1 < chars.length && isLower(chars[index + 1]);
};

// "PVCResizeFailed" reads "PVC resize failed". The same steps as everestctl
// takes, so that both spell a reason neither knows alike.
const splitWords = (name: string) => {
  const chars = [...name];

  return chars
    .map((char, index) => {
      if (index === 0 || !startsWord(chars, index)) {
        return char;
      }
      const lowered =
        index + 1 < chars.length && isLower(chars[index + 1])
          ? char.toLowerCase()
          : char;
      return ` ${lowered}`;
    })
    .join('');
};

// Never fails: providers add reasons of their own, and one this client has
// not heard of still has to be shown.
export const getReasonLabel = (reason: string) =>
  Object.prototype.hasOwnProperty.call(REASON_LABELS, reason)
    ? REASON_LABELS[reason]
    : splitWords(reason);

const toAvailability = (status: string | undefined): InstanceAvailability => {
  if (status === 'True') {
    return 'Yes';
  }
  return status === 'False' ? 'No' : 'Unknown';
};

export const isInstanceDeleting = (instance?: InstanceStatusSource) =>
  !!instance?.metadata?.deletionTimestamp;

// The phase is the only record of whether the database has ever been up.
const hasInstanceBeenAvailable = (instance?: InstanceStatusSource) =>
  instance?.status?.phase === PHASE_PROVISIONED;

// An Instance no provider has reconciled is stale too: its observed
// generation is the API default, below any generation.
const isStatusStale = (instance?: InstanceStatusSource) =>
  (instance?.status?.observedGeneration ?? 0) <
  (instance?.metadata?.generation ?? 0);

// The Unattended condition, if it is to be honoured: only while the status is
// stale, since a provider that has caught up has answered it, and only True
// or Unknown, the statuses core writes.
const getUnattended = (
  instance: InstanceStatusSource | undefined,
  stale: boolean
) => {
  const unattended = getInstanceCondition(
    instance,
    InstanceConditionType.Unattended
  );

  return stale &&
    (unattended?.status === 'True' || unattended?.status === 'Unknown')
    ? unattended
    : undefined;
};

// The first case that matches wins. A stall on an older spec says nothing
// about the current one: a fix may just have been pushed.
const getTone = (
  instance: InstanceStatusSource | undefined,
  stale: boolean
): InstanceTone => {
  const unattended = getUnattended(instance, stale);

  if (isInstanceDeleting(instance)) {
    return 'deleting';
  }
  if (unattended) {
    return unattended.status === 'True' ? 'attention' : 'unknown';
  }
  if (isInstanceConditionTrue(instance, InstanceConditionType.Suspended)) {
    return 'stopped';
  }
  if (stale) {
    return 'progress';
  }
  if (isInstanceConditionTrue(instance, InstanceConditionType.Stalled)) {
    return 'error';
  }
  if (isInstanceConditionTrue(instance, InstanceConditionType.NeedsAttention)) {
    return 'attention';
  }
  return isInstanceConditionTrue(instance, InstanceConditionType.Ready)
    ? 'ok'
    : 'progress';
};

// Turns an Instance's status into what a person is shown: a tone that says
// whether to wait or act, one status word, and whether the database is up.
//
// everestctl has the same function in Go (pkg/instancestatus), and both are
// held to the same fixtures. Nothing is inferred from how long the status
// has not changed: when nobody attends to an Instance, core says so.
export const getInstanceStatusDisplay = (
  instance: InstanceStatusSource | undefined
): InstanceStatusDisplay => {
  const ready = getInstanceCondition(instance, InstanceConditionType.Ready);
  const stale = isStatusStale(instance);
  const tone = getTone(instance, stale);
  const unattended = getUnattended(instance, stale);
  const phase = instance?.status?.phase;
  const neverUp = !phase || phase === PHASE_PENDING;

  let reason = neverUp ? REASON_CREATING : REASON_APPLYING_CHANGES;
  if (unattended?.reason) {
    reason = unattended.reason;
  } else if (ready?.reason && !stale) {
    reason = ready.reason;
  } else if (tone === 'deleting') {
    reason = REASON_DELETING;
  }

  return {
    tone,
    reason,
    label: getReasonLabel(reason),
    // Availability is observed, so it holds whichever spec the rest of the
    // status describes.
    available: toAvailability(
      getInstanceCondition(instance, InstanceConditionType.Available)?.status
    ),
    message: (unattended ? unattended.message : !stale && ready?.message) || '',
    stale,
  };
};

// Whether clients can connect is worth saying only while work is in flight on
// a database that has been up. Before that it is not up by definition, and
// outside of work the status word says it.
const getConnectivityDuringWork = (
  instance: InstanceStatusSource | undefined,
  { tone, available }: Pick<InstanceStatusDisplay, 'tone' | 'available'>
): Exclude<InstanceAvailability, 'Unknown'> | undefined =>
  tone === 'progress' &&
  hasInstanceBeenAvailable(instance) &&
  available !== 'Unknown'
    ? available
    : undefined;

// What a status badge says: the status word, with what the word alone leaves
// out.
export const getInstanceStatusText = (
  instance: InstanceStatusSource | undefined,
  status: InstanceStatusDisplay
) => {
  if (status.tone === 'deleting') {
    const deleting = getReasonLabel(REASON_DELETING);
    const ready = getInstanceCondition(instance, InstanceConditionType.Ready);
    // The status word says what the deletion is waiting for, unless it dates
    // from before the deletion: a database that was ready is being deleted,
    // not ready.
    const predatesDeletion =
      ready?.status === 'True' && ready.reason === status.reason;

    return status.reason === REASON_DELETING || predatesDeletion
      ? deleting
      : Messages.withDetail(deleting, status.label);
  }

  const connectivity = getConnectivityDuringWork(instance, status);

  return connectivity
    ? Messages.withDetail(status.label, Messages.connectivity[connectivity])
    : status.label;
};

// What is wrong with the Instance, if its status says something is. The
// problem is whatever NeedsAttention carries, which during a named operation
// is not the status word: an upgrade whose pods crash reads "Upgrading".
export const getInstanceProblem = (
  instance: InstanceStatusSource | undefined
): InstanceProblem | undefined => {
  const status = getInstanceStatusDisplay(instance);

  if (status.tone !== 'attention' && status.tone !== 'error') {
    return undefined;
  }

  // While stale, the only problem known is the one core reports: what
  // NeedsAttention says describes an older spec.
  const needsAttention = status.stale
    ? undefined
    : getInstanceCondition(instance, InstanceConditionType.NeedsAttention);
  const reason = needsAttention?.reason || status.reason;
  const outlook = status.tone === 'error' ? 'stalled' : 'retrying';

  return {
    reason,
    label: getReasonLabel(reason),
    message: needsAttention?.message || status.message,
    outlook: status.stale ? 'unattended' : outlook,
  };
};

// What the UI allows depends on the condition that makes it possible, never on
// the status word or its tone: those are for people.

// Connection details are published once and outlive an upgrade.
export const hasConnectionDetails = (instance?: InstanceStatusSource) =>
  isInstanceConditionTrue(
    instance,
    InstanceConditionType.ConnectionDetailsReady
  );

export const isInstanceRestoring = (instance?: InstanceStatusSource) =>
  isInstanceConditionTrue(instance, InstanceConditionType.Restoring);

// The operations a spec change has to wait for. Converging without a named
// operation is not one of them.
const EDIT_BLOCKING_OPERATIONS = [
  InstanceConditionType.Restoring,
  InstanceConditionType.Upgrading,
  InstanceConditionType.StorageResizing,
];

// An Instance in trouble stays editable: editing it is how it gets fixed.
// One no provider has reported on is held back, as it was while it had no
// phase: every provider report carries Ready. It can still be deleted.
export const areInstanceEditsBlocked = (instance?: InstanceStatusSource) =>
  isInstanceDeleting(instance) ||
  EDIT_BLOCKING_OPERATIONS.some((operation) =>
    isInstanceConditionTrue(instance, operation)
  ) ||
  !getInstanceCondition(instance, InstanceConditionType.Ready);
