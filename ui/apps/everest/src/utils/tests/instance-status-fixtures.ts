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

// The cases everestctl's implementation of the status display runs too: one
// file, at the repository root, keeps the two from drifting apart.
import fixtures from '../../../../../../pkg/instancestatus/testdata/display.json';
import { Instance } from 'shared-types/api.types';
import {
  InstanceCondition,
  InstanceStatusDisplay,
  InstanceStatusSource,
} from 'shared-types/instance.types';

export interface InstanceStatusFixture {
  name: string;
  about: string;
  now: string;
  instance: {
    metadata: NonNullable<InstanceStatusSource['metadata']> & {
      name: string;
      namespace: string;
    };
    status: NonNullable<Instance['status']>;
  };
  // The fields the UI is held to. The file also records what other clients
  // make of the same object.
  want: Pick<
    InstanceStatusDisplay,
    'tone' | 'reason' | 'label' | 'available' | 'sentence' | 'since'
  >;
}

export const INSTANCE_STATUS_FIXTURES =
  fixtures.cases as unknown as InstanceStatusFixture[];

export const INSTANCE_STATUS_FIXTURE_LABELS: Record<string, string> =
  fixtures.labels;

export const getInstanceStatusFixture = (name: string) => {
  const fixture = INSTANCE_STATUS_FIXTURES.find((item) => item.name === name);
  if (!fixture) {
    throw new Error(`no status fixture named ${name}`);
  }
  return fixture;
};

// An Instance in the state a fixture describes, with enough around the
// status to render it where a whole Instance is expected.
export const getFixtureInstance = (
  name: string,
  overrides: Partial<Instance> = {}
): Instance => {
  const { instance } = getInstanceStatusFixture(name);

  return {
    apiVersion: 'core.openeverest.io/v1alpha1',
    kind: 'Instance',
    metadata: instance.metadata,
    spec: { providerRef: { name: 'percona-server-mongodb' } },
    status: instance.status,
    ...overrides,
  };
};

type InstanceStatus = NonNullable<Instance['status']>;

export type FixtureCondition = Pick<
  InstanceCondition,
  'type' | 'status' | 'reason'
> &
  Partial<InstanceCondition>;

// The fixtures carry what the status word depends on. This adds the rest of
// what a real status holds: more conditions, and the components.
export const withStatus = (
  instance: Instance,
  conditions: FixtureCondition[],
  components: InstanceStatus['components'] = instance.status?.components
): Instance => ({
  ...instance,
  status: {
    ...instance.status,
    components,
    conditions: [
      ...(instance.status?.conditions ?? []),
      ...conditions.map((condition) => ({
        message: '',
        lastTransitionTime: '2026-03-01T11:00:00Z',
        ...condition,
      })),
    ],
  },
});
