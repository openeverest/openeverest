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

import { InstanceStatusSource } from 'shared-types/instance.types';
import { isInstanceDeleting } from './instance-status';
import {
  areInstanceEditsBlocked,
  hasConnectionDetails,
  isInstanceRestoring,
} from './instance-gating';
import { HEALTHY, condition, instance } from './instance-status.test-utils';

describe('action gating', () => {
  it.each<[string, InstanceStatusSource | undefined, boolean]>([
    ['healthy', instance(HEALTHY), false],
    [
      'being deleted',
      instance(HEALTHY, { deletionTimestamp: '2026-03-01T12:00:00Z' }),
      true,
    ],
    ['restoring', instance([...HEALTHY, condition('Restoring', 'True')]), true],
    ['upgrading', instance([...HEALTHY, condition('Upgrading', 'True')]), true],
    [
      'resizing its storage',
      instance([...HEALTHY, condition('StorageResizing', 'True')]),
      true,
    ],
    [
      'restore finished',
      instance([...HEALTHY, condition('Restoring', 'False')]),
      false,
    ],
    [
      'converging without a named operation',
      instance([
        condition('Available', 'True'),
        condition('Ready', 'False', 'ApplyingChanges'),
        condition('Reconciling', 'True'),
      ]),
      false,
    ],
    [
      'stalled, so that it can be fixed',
      instance([
        condition('Ready', 'False', 'ValidationFailed'),
        condition('Stalled', 'True'),
        condition('NeedsAttention', 'True'),
      ]),
      false,
    ],
    ['not reported on by a provider', instance([]), true],
    [
      'attended by nobody',
      instance([condition('Unattended', 'True', 'ProviderNotFound')]),
      true,
    ],
    ['not loaded', undefined, true],
  ])('blocks edits of an Instance that is %s: %s', (_, source, blocked) => {
    expect(areInstanceEditsBlocked(source)).toBe(blocked);
  });

  it('leaves an Instance no provider has reported on deletable', () => {
    const neverReconciled: InstanceStatusSource = {
      metadata: { generation: 1 },
      status: { phase: 'Pending', observedGeneration: -1 },
    };

    expect(areInstanceEditsBlocked(neverReconciled)).toBe(true);
    // Deleting is held back only for an Instance already being deleted.
    expect(isInstanceDeleting(neverReconciled)).toBe(false);
  });

  it('tells deletion from the deletion timestamp, not from the phase', () => {
    expect(
      isInstanceDeleting({
        metadata: {},
        status: { phase: 'Terminating' },
      })
    ).toBe(false);
    expect(
      isInstanceDeleting(
        instance(HEALTHY, { deletionTimestamp: '2026-03-01T12:00:00Z' })
      )
    ).toBe(true);
  });

  it('keeps connection details through an upgrade', () => {
    const upgrading = instance([
      condition('Available', 'True'),
      condition('Ready', 'False', 'Upgrading'),
      condition('Upgrading', 'True'),
      condition('ConnectionDetailsReady', 'True'),
    ]);

    expect(hasConnectionDetails(upgrading)).toBe(true);
    expect(hasConnectionDetails(instance([condition('Ready', 'True')]))).toBe(
      false
    );
  });

  it('reads a running restore from its own condition', () => {
    expect(
      isInstanceRestoring(instance([condition('Restoring', 'True')]))
    ).toBe(true);
    expect(
      isInstanceRestoring(instance([condition('Ready', 'False', 'Restoring')]))
    ).toBe(false);
  });
});
