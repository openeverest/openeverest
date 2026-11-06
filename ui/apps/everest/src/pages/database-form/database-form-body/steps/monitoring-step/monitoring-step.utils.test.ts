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
  InstanceMonitoringDestination,
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import {
  DestinationDisabledReason,
  DestinationReadiness,
} from './monitoring-step.types';
import {
  buildInstanceFormsByDestination,
  buildMonitoringParametersSchema,
  finalizeMonitoringEntries,
  getDefaultEntryParameters,
  getDestinationDisabledReason,
  getDestinationReadiness,
  getDestinationSummary,
  groupDestinationsByClass,
  parseMonitoringDestinations,
  pruneMonitoringDestinations,
  toggleMonitoringDestination,
  toMonitoringEntryName,
  writeMonitoringEntries,
} from './monitoring-step.utils';

const PROVIDER = 'percona-server-mongodb';

const makeClass = (
  name: string,
  mode: 'ProviderManaged' | 'ExtensionManaged',
  integration?: string
): MonitoringClass => ({
  metadata: { name },
  spec: {
    authModes: [{ name: 'Anonymous' }],
    ...(integration && {
      providerManaged: {
        integration,
        agentImage: 'agent:1',
        agentVersion: '1.0.0',
      },
    }),
  },
  status: { providers: [{ name: PROVIDER, mode }] },
});

const makeDestination = (
  name: string,
  className: string,
  conditions: { type: string; status: 'True' | 'False' | 'Unknown' }[] = [],
  parameters?: Record<string, string>
): MonitoringDestination => {
  const destination: MonitoringDestination = {
    metadata: { name },
    spec: { classRef: { name: className } },
    status: {
      conditions: conditions.map((condition) => ({
        ...condition,
        reason: 'Test',
        message: '',
        lastTransitionTime: '2026-01-01T00:00:00Z',
      })),
    },
  };
  if (parameters) {
    Reflect.set(destination.spec, 'parameters', parameters);
  }
  return destination;
};

const entry = (
  name: string,
  destinationName = name
): InstanceMonitoringDestination => ({
  name,
  destinationRef: { name: destinationName },
});

const pmm = makeClass('pmm', 'ProviderManaged', 'pmm');
const prometheus = makeClass('prometheus', 'ExtensionManaged');

const pmmWithInstanceForm = makeClass('pmm', 'ProviderManaged', 'pmm');
Reflect.set(pmmWithInstanceForm.spec, 'uiSchema', {
  instance: {
    components: {
      cpuRequest: {
        path: 'parameters.resources.requests.cpu',
        uiType: 'text',
        fieldParams: { label: 'CPU request', defaultValue: '100m' },
      },
      replicas: {
        path: 'parameters.replicas',
        uiType: 'number',
        fieldParams: { label: 'Replicas' },
        validation: { min: 1 },
      },
    },
  },
});

const instanceForms = buildInstanceFormsByDestination(
  [pmmWithInstanceForm, prometheus],
  [makeDestination('pmm-a', 'pmm'), makeDestination('prom', 'prometheus')]
);

describe('toMonitoringEntryName', () => {
  it('uses the destination name when it is a free DNS label', () => {
    expect(toMonitoringEntryName('prometheus', [])).toBe('prometheus');
  });

  it('turns a dotted destination name into a DNS label', () => {
    expect(toMonitoringEntryName('pmm.prod.eu', [])).toBe('pmm-prod-eu');
  });

  it('adds a numeric suffix when the name is taken', () => {
    expect(toMonitoringEntryName('a.b', ['a-b'])).toBe('a-b-2');
    expect(toMonitoringEntryName('a.b', ['a-b', 'a-b-2'])).toBe('a-b-3');
  });

  it('keeps suffixed names within 63 characters', () => {
    const long = 'x'.repeat(70);
    const first = toMonitoringEntryName(long, []);
    const second = toMonitoringEntryName(long, [first]);

    expect(first).toHaveLength(63);
    expect(second).toHaveLength(63);
    expect(second.endsWith('-2')).toBe(true);
  });
});

describe('toggleMonitoringDestination', () => {
  it('adds an entry referencing the destination', () => {
    expect(toggleMonitoringDestination([], 'prometheus')).toEqual([
      entry('prometheus'),
    ]);
  });

  it('removes the entry of a selected destination', () => {
    expect(
      toggleMonitoringDestination(
        [entry('pmm'), entry('prometheus')],
        'prometheus'
      )
    ).toEqual([entry('pmm')]);
  });
});

describe('buildInstanceFormsByDestination', () => {
  it('maps only destinations whose class defines an instance form', () => {
    expect([...instanceForms.keys()]).toEqual(['pmm-a']);
  });

  it('seeds a new entry with the form defaults', () => {
    const defaults = getDefaultEntryParameters(instanceForms.get('pmm-a'));

    expect(toggleMonitoringDestination([], 'pmm-a', defaults)).toEqual([
      {
        ...entry('pmm-a'),
        parameters: expect.objectContaining({
          resources: { requests: { cpu: '100m' } },
        }),
      },
    ]);
    expect(getDefaultEntryParameters(undefined)).toBeUndefined();
  });
});

describe('buildMonitoringParametersSchema', () => {
  const schema = buildMonitoringParametersSchema(instanceForms);
  const formWith = (destinations: unknown[]) => ({
    monitoring: { destinations },
  });

  it('reports an invalid parameter at its entry', () => {
    const result = schema.safeParse(
      formWith([
        entry('prom'),
        { ...entry('pmm-a'), parameters: { replicas: 0 } },
      ])
    );

    expect(result.success).toBe(false);
    expect(result.error?.issues.map((issue) => issue.path)).toEqual([
      ['monitoring', 'destinations', 1, 'parameters', 'replicas'],
    ]);
  });

  it('accepts entries without a form or with valid parameters', () => {
    expect(
      schema.safeParse(
        formWith([
          entry('prom'),
          { ...entry('pmm-a'), parameters: { replicas: 2 } },
        ])
      ).success
    ).toBe(true);
    expect(schema.safeParse({}).success).toBe(true);
  });
});

describe('finalizeMonitoringEntries', () => {
  it('cleans parameters and coerces numbers of the class form', () => {
    const [pmmEntry] = parseMonitoringDestinations([
      {
        ...entry('pmm-a'),
        parameters: {
          replicas: '2',
          resources: { requests: { cpu: '' } },
        },
      },
    ]);

    expect(finalizeMonitoringEntries([pmmEntry], instanceForms)).toEqual([
      { ...entry('pmm-a'), parameters: { replicas: 2 } },
    ]);
  });

  it('omits parameters when nothing was filled', () => {
    const entries = parseMonitoringDestinations([
      {
        ...entry('pmm-a'),
        parameters: { resources: { requests: { cpu: '' } } },
      },
      { ...entry('prom'), parameters: {} },
    ]);

    expect(finalizeMonitoringEntries(entries, instanceForms)).toEqual([
      entry('pmm-a'),
      entry('prom'),
    ]);
  });
});

describe('writeMonitoringEntries', () => {
  it('writes the finalized entries into the request', () => {
    const formData = {
      monitoring: {
        destinations: [{ ...entry('pmm-a'), parameters: { replicas: '2' } }],
      },
    };

    writeMonitoringEntries(formData, instanceForms);

    expect(formData).toEqual({
      monitoring: {
        destinations: [{ ...entry('pmm-a'), parameters: { replicas: 2 } }],
      },
    });
  });

  it('drops spec.monitoring when no destination is selected', () => {
    const formData = { dbName: 'db', monitoring: { destinations: [] } };

    writeMonitoringEntries(formData, instanceForms);

    expect(formData).toEqual({ dbName: 'db' });
  });
});

describe('getDestinationReadiness', () => {
  it('is ready when Ready is True', () => {
    expect(
      getDestinationReadiness(
        makeDestination('d', 'prometheus', [
          { type: 'Accepted', status: 'True' },
          { type: 'Ready', status: 'True' },
        ])
      )
    ).toBe(DestinationReadiness.Ready);
  });

  it('is not ready when core did not accept the destination', () => {
    expect(
      getDestinationReadiness(
        makeDestination('d', 'prometheus', [
          { type: 'Accepted', status: 'False' },
          { type: 'Ready', status: 'Unknown' },
        ])
      )
    ).toBe(DestinationReadiness.NotReady);
  });

  it('is pending while its controller has not reported', () => {
    expect(
      getDestinationReadiness(
        makeDestination('d', 'prometheus', [
          { type: 'Ready', status: 'Unknown' },
        ])
      )
    ).toBe(DestinationReadiness.Pending);
  });
});

describe('getDestinationSummary', () => {
  it('returns the url parameter', () => {
    expect(
      getDestinationSummary(
        makeDestination('d', 'pmm', [], { url: 'https://pmm.example.com' })
      )
    ).toBe('https://pmm.example.com');
  });

  it('returns nothing for a class without a url parameter', () => {
    expect(
      getDestinationSummary(
        makeDestination('d', 'prometheus', [], { interval: '30s' })
      )
    ).toBeUndefined();
  });
});

describe('groupDestinationsByClass', () => {
  it('keeps class order and lists classes without destinations', () => {
    const groups = groupDestinationsByClass(
      [pmm, prometheus],
      [makeDestination('prom', 'prometheus')]
    );

    expect(groups.map((g) => g.monitoringClass.metadata?.name)).toEqual([
      'pmm',
      'prometheus',
    ]);
    expect(groups[0].destinations).toEqual([]);
    expect(groups[1].destinations.map((d) => d.metadata?.name)).toEqual([
      'prom',
    ]);
  });
});

describe('getDestinationDisabledReason', () => {
  const pmmA = makeDestination('pmm-a', 'pmm');
  const pmmB = makeDestination('pmm-b', 'pmm');
  const promA = makeDestination('prom-a', 'prometheus');
  const promB = makeDestination('prom-b', 'prometheus');
  const destinations = [pmmA, pmmB, promA, promB];
  const classes = [pmm, prometheus];

  const reasonFor = (
    destination: MonitoringDestination,
    selected: InstanceMonitoringDestination[]
  ) =>
    getDestinationDisabledReason({
      destination,
      selected,
      destinations,
      classes,
      provider: PROVIDER,
    });

  it('allows a second destination of an extension-managed class', () => {
    expect(reasonFor(promB, [entry('prom-a')])).toBeUndefined();
  });

  it('blocks a second destination of the same provider-managed integration', () => {
    expect(reasonFor(pmmB, [entry('pmm-a')])).toBe(
      DestinationDisabledReason.IntegrationTaken
    );
  });

  it('allows mixing a provider-managed and an extension-managed class', () => {
    expect(reasonFor(promA, [entry('pmm-a')])).toBeUndefined();
    expect(reasonFor(pmmA, [entry('prom-a')])).toBeUndefined();
  });

  it('applies the integration rule only when the class is provider-managed for this provider', () => {
    const extensionManagedPmm: MonitoringClass = {
      ...pmm,
      status: { providers: [{ name: PROVIDER, mode: 'ExtensionManaged' }] },
    };

    expect(
      getDestinationDisabledReason({
        destination: pmmB,
        selected: [entry('pmm-a')],
        destinations,
        classes: [extensionManagedPmm, prometheus],
        provider: PROVIDER,
      })
    ).toBeUndefined();
  });

  it('never disables a selected destination, so it can be deselected', () => {
    expect(reasonFor(pmmA, [entry('pmm-a')])).toBeUndefined();
  });

  it('blocks a ninth destination', () => {
    const eight = Array.from({ length: 8 }, (_, i) => entry(`e${i}`));

    expect(reasonFor(promA, eight)).toBe(
      DestinationDisabledReason.LimitReached
    );
  });
});

describe('pruneMonitoringDestinations', () => {
  it('drops entries whose destination is not offered', () => {
    expect(
      pruneMonitoringDestinations(
        [entry('pmm'), entry('prometheus')],
        ['prometheus']
      )
    ).toEqual([entry('prometheus')]);
  });
});

describe('parseMonitoringDestinations', () => {
  it('keeps only well-formed entries', () => {
    expect(
      parseMonitoringDestinations([
        entry('prometheus'),
        { name: 'broken' },
        'nope',
      ])
    ).toEqual([entry('prometheus')]);
  });

  it('returns an empty list for a non-array value', () => {
    expect(parseMonitoringDestinations(undefined)).toEqual([]);
  });
});
