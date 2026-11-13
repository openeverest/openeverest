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

// Renders the three places an Instance's status is shown (the databases
// list, the header of the details page and the Status card) from the status
// fixtures shared with everestctl, in a real browser and in both themes.
//
// Besides what it asserts, each case saves a picture under
// test-results/instance-status/, so the result can be looked at without a
// cluster.

import { render, screen, waitFor } from '@testing-library/react';
import { ThemeContextProvider, everestThemeOptions } from '@percona/design';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { page, userEvent } from 'vitest/browser';
import { DbInstanceForNamespaceResult } from 'hooks/api/db-instances';
import { NowContext } from 'hooks/utils/useNow';
import { DbClusterView } from 'pages/databases/DbClusterView';
import { convertDbInstancesPayloadToTableFormat } from 'pages/databases/DbClusterView.utils';
import { useDatabasesView } from 'pages/databases/hooks/useDatabasesView';
import { StatusCard } from 'pages/db-cluster-details/cluster-overview/sections/status-card';
import { DbClusterDetails } from 'pages/db-cluster-details/db-cluster-details';
import { DbInstanceContext } from 'pages/db-cluster-details/dbCluster.context';
import { Instance } from 'shared-types/api.types';
import {
  FixtureCondition,
  getFixtureInstance,
  INSTANCE_STATUS_FIXTURES,
  withStatus,
} from 'utils/tests/instance-status-fixtures';

vi.mock('pages/databases/hooks/useDatabasesView', () => ({
  useDatabasesView: vi.fn(),
}));

vi.mock('contexts/plugins', () => ({
  usePlugins: () => ({ plugins: [] }),
}));

// The real menus load backups and permissions; a button of the same shape
// stands in for them.
vi.mock('components/db-actions/db-actions', async () => {
  const { Button, IconButton } = await import('@mui/material');
  const { default: MoreHorizIcon } = await import(
    '@mui/icons-material/MoreHoriz'
  );
  const { default: ArrowDropDownIcon } = await import(
    '@mui/icons-material/ArrowDropDown'
  );
  const DbActions = ({ showStatusActions }: { showStatusActions?: boolean }) =>
    showStatusActions ? (
      <Button variant="text" size="large" endIcon={<ArrowDropDownIcon />}>
        Actions
      </Button>
    ) : (
      <IconButton>
        <MoreHorizIcon />
      </IconButton>
    );

  return { DbActions, default: DbActions };
});

// Every fixture is taken at the same moment.
const NOW = new Date(INSTANCE_STATUS_FIXTURES[0].now);
const PAGE_WIDTH = 1280;
// Tall enough for the longest surface: what is below the fold is not in the
// picture.
const PAGE_HEIGHT = 1600;
const THEMES = ['light', 'dark'] as const;
type Theme = (typeof THEMES)[number];

const ago = (minutes: number) =>
  new Date(NOW.getTime() - minutes * 60 * 1000).toISOString();

const passing = (type: string, reason: string): FixtureCondition => ({
  type,
  status: 'True',
  reason,
  lastTransitionTime: ago(40 * 24 * 60),
});

const starting: FixtureCondition = {
  type: 'PodsReady',
  status: 'Unknown',
  reason: 'NotReady',
  message: 'engine: 3 pods are not ready',
  lastTransitionTime: ago(2),
};

// What a provider reports beside the conditions a fixture is about, so that
// the Status card has every section to show. A database that has been up
// reports every check; these are the ones that have not.
const BEFORE_FIRST_UP: Record<
  string,
  { conditions: FixtureCondition[]; ready?: number }
> = {
  creating: {
    conditions: [
      passing('ReferencesResolved', 'Resolved'),
      passing('Synced', 'Synced'),
      passing('PodsScheduled', 'Scheduled'),
      starting,
    ],
    ready: 0,
  },
  'seeding-from-backup': {
    conditions: [
      passing('ReferencesResolved', 'Resolved'),
      passing('Synced', 'Synced'),
      passing('PodsScheduled', 'Scheduled'),
      passing('PodsReady', 'Ready'),
      passing('EngineHealthy', 'Healthy'),
    ],
    ready: 3,
  },
  // The provider turned the spec down, so there is nothing to check yet.
  'rejected-spec': { conditions: [passing('ReferencesResolved', 'Resolved')] },
};

const reported = (fixture: string): Instance => {
  const instance = getFixtureInstance(fixture);
  const has = (type: string) =>
    !!instance.status?.conditions?.some((item) => item.type === type);
  const unless = (type: string, reason: string) =>
    has(type) ? [] : [passing(type, reason)];
  const components = (ready: number) => [
    { name: 'engine', replicas: 3, readyReplicas: ready },
    { name: 'proxy', replicas: 2, readyReplicas: ready && 2 },
  ];

  if (instance.status?.phase !== 'Provisioned') {
    const before = BEFORE_FIRST_UP[fixture];
    return before
      ? withStatus(
          instance,
          before.conditions,
          before.ready === undefined ? undefined : components(before.ready)
        )
      : instance;
  }

  return withStatus(
    instance,
    [
      passing('ReferencesResolved', 'Resolved'),
      passing('Synced', 'Synced'),
      passing('PodsScheduled', 'Scheduled'),
      ...unless('PodsReady', 'Ready'),
      passing('EngineHealthy', 'Healthy'),
      ...unless('BackupConfigured', 'Configured'),
      passing('ConnectionDetailsReady', 'Published'),
    ],
    // A pod that is not ready is behind every fixture that mentions pods.
    components(has('PodsReady') ? 2 : 3)
  );
};

// The upgrade, with something that finished before it and something that
// only wants a decision: the card with every section filled.
const upgradeWithHistory = () =>
  withStatus(reported('rolling-upgrade'), [
    {
      type: 'StorageResizing',
      status: 'False',
      reason: 'Succeeded',
      message: 'Volumes expanded to 50Gi',
      lastTransitionTime: ago(180),
    },
    {
      type: 'MaintenancePending',
      status: 'True',
      reason: 'AwaitingApproval',
      message: '1 action awaits approval',
      lastTransitionTime: ago(60),
    },
  ]);

const named = (
  instance: Instance,
  name: string,
  provider: string,
  namespace = 'demo'
): Instance => ({
  ...instance,
  metadata: { ...instance.metadata, name, namespace },
  spec: { ...instance.spec, providerRef: { name: provider } },
});

// A fleet with one database in each state a list has to tell apart.
const FLEET: Instance[] = [
  named(reported('healthy'), 'orders', 'percona-server-mongodb', 'production'),
  named(reported('deleting'), 'old', 'percona-server-mongodb'),
  named(reported('stopped'), 'sandbox', 'percona-postgresql'),
  named(reported('change-availability-unknown'), 'cache', 'valkey'),
  named(reported('creating'), 'staging', 'percona-xtradb-cluster'),
  named(upgradeWithHistory(), 'billing', 'percona-postgresql', 'production'),
  named(reported('in-place-restore'), 'reports', 'percona-server-mongodb'),
  named(
    reported('backup-storage-missing-early'),
    'analytics',
    'percona-postgresql',
    'production'
  ),
  named(reported('rejected-spec'), 'new-app', 'percona-postgresql'),
  named(
    reported('crash-loop-all-replicas-stalled'),
    'legacy',
    'percona-xtradb-cluster',
    'production'
  ),
];

const Surface = ({
  theme,
  children,
}: {
  theme: Theme;
  children: React.ReactNode;
}) => {
  localStorage.setItem('colorMode', theme);

  return (
    <ThemeContextProvider
      themeOptions={everestThemeOptions}
      saveColorModeOnLocalStorage
    >
      <NowContext.Provider value={NOW}>
        <div
          data-testid="surface"
          style={{
            width: PAGE_WIDTH,
            padding: 24,
            boxSizing: 'border-box',
            background: theme === 'dark' ? '#121212' : '#ffffff',
          }}
        >
          {children}
        </div>
      </NowContext.Provider>
    </ThemeContextProvider>
  );
};

const renderList = (theme: Theme, instances: Instance[]) => {
  const byNamespace = new Map<string, Instance[]>();
  instances.forEach((instance) => {
    const namespace = instance.metadata?.namespace ?? '';
    byNamespace.set(namespace, [
      ...(byNamespace.get(namespace) ?? []),
      instance,
    ]);
  });
  vi.mocked(useDatabasesView).mockReturnValue({
    state: {
      status: 'table',
      rows: convertDbInstancesPayloadToTableFormat(
        [...byNamespace].map(
          ([namespace, data]) =>
            ({
              namespace,
              queryResult: { data, isSuccess: true },
            }) as DbInstanceForNamespaceResult
        ),
        NOW
      ),
    },
    namespaces: [...byNamespace.keys()],
    providersNamesFilter: [],
    hasCreatePermission: false,
    canAddCluster: false,
  });

  return render(
    <MemoryRouter>
      <Surface theme={theme}>
        <DbClusterView />
      </Surface>
    </MemoryRouter>
  );
};

// The details page of a database, with the Status card where the Overview
// tab puts it.
const renderDetails = (theme: Theme, instance: Instance, withCard = true) =>
  render(
    <MemoryRouter initialEntries={['/databases/demo/db/overview']}>
      <Surface theme={theme}>
        <Routes>
          <Route
            path="/databases/:namespace/:instanceName"
            element={
              <DbInstanceContext.Provider
                value={{
                  instance,
                  isLoading: false,
                  instanceDeleted: false,
                  canReadCredentials: true,
                  isCredentialsPermissionLoading: false,
                }}
              >
                <DbClusterDetails />
              </DbInstanceContext.Provider>
            }
          >
            <Route
              path="overview"
              element={withCard ? <StatusCard instance={instance} /> : null}
            />
          </Route>
        </Routes>
      </Surface>
    </MemoryRouter>
  );

const shoot = async (name: string) => {
  const surface = screen.getByTestId('surface');
  await document.fonts.ready;
  // Off whatever the pointer last clicked, so that nothing is highlighted.
  await userEvent.hover(surface, { position: { x: 1, y: 1 } });
  await page.screenshot({
    element: surface,
    path: `../../../test-results/instance-status/${name}.png`,
  });
};

// The names of the rows, top to bottom.
const listedNames = () =>
  screen
    .getAllByTestId(/-available$/)
    .map((cell) =>
      cell.getAttribute('data-testid')!.replace(/-available$/, '')
    );

// A label that does not fit its cell is cut with an ellipsis.
const isCut = (element: Element) => {
  const cell = element.closest('td') ?? element;
  return cell.scrollWidth > cell.clientWidth;
};

const cutHeaders = () =>
  screen
    .getAllByRole('columnheader')
    .flatMap((header) => [
      ...header.querySelectorAll('.Mui-TableHeadCell-Content-Wrapper'),
    ])
    .filter(isCut)
    .map((header) => header.textContent);

beforeAll(async () => {
  await page.viewport(PAGE_WIDTH, PAGE_HEIGHT);
});

describe.each(THEMES)('Instance status, %s theme', (theme) => {
  it('lists a fleet with the databases that need someone first', async () => {
    renderList(theme, FLEET);

    await waitFor(() =>
      expect(listedNames()).toEqual([
        'legacy',
        'new-app',
        'analytics',
        'staging',
        'reports',
        'cache',
        'billing',
        'sandbox',
        'old',
        'orders',
      ])
    );
    expect(screen.getByTestId('legacy-status')).toHaveTextContent('Crash loop');
    expect(screen.getByTestId('legacy-available')).toHaveTextContent('No');
    expect(screen.getByTestId('analytics-status')).toHaveTextContent(
      'Backup storage not found'
    );
    expect(screen.getByTestId('analytics-available')).toHaveTextContent('Yes');
    expect(screen.getByTestId('cache-available')).toHaveTextContent('\u2014');
    expect(screen.getByTestId('billing-notices')).toHaveTextContent('1 notice');
    expect(screen.getByTestId('needs-attention-filter')).toHaveTextContent(
      'Needs attention (3)'
    );
    expect(
      screen.getAllByTestId(/-(status|available|notices)$/).filter(isCut)
    ).toEqual([]);
    expect(cutHeaders()).toEqual([]);

    await shoot(`list-${theme}`);
  });

  it('narrows the fleet to what needs attention', async () => {
    renderList(theme, FLEET);
    await waitFor(() => expect(listedNames()).toHaveLength(FLEET.length));

    await userEvent.click(screen.getByTestId('needs-attention-filter'));

    await waitFor(() =>
      expect(listedNames()).toEqual(['legacy', 'new-app', 'analytics'])
    );

    await shoot(`list-needs-attention-${theme}`);
  });

  it.each([
    ['rolling-upgrade', 'Up \u00b7 Upgrading', false],
    ['healthy', 'Up \u00b7 Ready', false],
    ['crash-loop-one-replica', 'Up \u00b7 Crash loop', true],
    ['crash-loop-all-replicas-stalled', 'Down \u00b7 Crash loop', true],
    ['secret-missing-early', 'Secret not found', true],
    [
      'deleting-waiting-for-children',
      'Deleting \u00b7 Waiting for backups',
      false,
    ],
  ])(
    'heads the details of the %s database with "%s"',
    async (fixture, sentence, alerts) => {
      renderDetails(theme, reported(fixture), false);

      expect(screen.getByTestId('db-status')).toHaveTextContent(sentence);
      expect(screen.queryAllByTestId('instance-status-alert')).toHaveLength(
        alerts ? 1 : 0
      );

      await shoot(`header-${fixture}-${theme}`);
    }
  );

  it.each([
    'healthy',
    'creating',
    'seeding-from-backup',
    'crash-loop-one-replica',
    'backup-storage-missing-stalled',
    'rejected-spec',
    'spec-edited-not-observed',
    'never-reconciled-fresh',
    'never-reconciled-old',
    'stopped',
  ])('fills the Status card of the %s database', async (fixture) => {
    render(
      <MemoryRouter>
        <Surface theme={theme}>
          <StatusCard instance={reported(fixture)} />
        </Surface>
      </MemoryRouter>
    );

    expect(screen.getByTestId('status-card-status')).toBeVisible();

    await shoot(`status-card-${fixture}-${theme}`);
  });

  it('fills every section of the Status card during an upgrade', async () => {
    render(
      <MemoryRouter>
        <Surface theme={theme}>
          <StatusCard instance={upgradeWithHistory()} />
        </Surface>
      </MemoryRouter>
    );

    expect(screen.getByTestId('status-card-status')).toHaveTextContent(
      'Up \u00b7 Upgrading'
    );
    expect(screen.getByTestId('status-component-engine')).toHaveTextContent(
      'engine 2/3 ready'
    );
    expect(screen.getByTestId('status-checks-passing')).toHaveTextContent(
      '6 checks passing'
    );
    expect(screen.getByTestId('status-check-PodsReady')).toBeVisible();
    expect(screen.getByTestId('status-operation-Upgrading')).toBeVisible();
    expect(
      screen.getByTestId('status-operation-StorageResizing')
    ).toBeVisible();
    expect(
      screen.getByTestId('status-notice-MaintenancePending')
    ).toBeVisible();

    await shoot(`status-card-rolling-upgrade-${theme}`);

    await userEvent.click(screen.getByTestId('status-checks-toggle'));
    await userEvent.click(screen.getByTestId('status-conditions-toggle'));
    await waitFor(() =>
      expect(screen.getByTestId('status-conditions-table')).toBeVisible()
    );
    // Wide enough for its six columns without scrolling sideways.
    const table = screen.getByTestId('status-conditions-table');
    expect(table.parentElement!.scrollWidth).toBeLessThanOrEqual(
      table.parentElement!.clientWidth
    );

    await shoot(`status-card-rolling-upgrade-expanded-${theme}`);
  });

  it.each(['rolling-upgrade', 'crash-loop-one-replica'])(
    'opens the details of the %s database with its status',
    async (fixture) => {
      renderDetails(
        theme,
        fixture === 'rolling-upgrade' ? upgradeWithHistory() : reported(fixture)
      );

      expect(screen.getByTestId('status-details')).toBeVisible();

      await shoot(`details-${fixture}-${theme}`);
    }
  );
});

describe('Instance status, every fixture', () => {
  // The table pages after ten rows.
  const pages = Array.from(
    { length: Math.ceil(INSTANCE_STATUS_FIXTURES.length / 10) },
    (_, index) => INSTANCE_STATUS_FIXTURES.slice(index * 10, index * 10 + 10)
  );

  it.each(pages.map((fixtures, index) => [index + 1, fixtures] as const))(
    'lists what each fixture expects, page %i',
    async (number, fixtures) => {
      renderList(
        'light',
        fixtures.map(({ name }) =>
          named(getFixtureInstance(name), name, 'percona-server-mongodb')
        )
      );
      await waitFor(() => expect(listedNames()).toHaveLength(fixtures.length));

      fixtures.forEach(({ name, want }) => {
        expect(screen.getByTestId(`${name}-status`)).toHaveTextContent(
          want.label
        );
        expect(screen.getByTestId(`${name}-available`)).toHaveTextContent(
          want.available === 'Unknown' ? '\u2014' : want.available
        );
      });
      expect(
        screen.getAllByTestId(/-(status|available)$/).filter(isCut)
      ).toEqual([]);

      await shoot(`list-fixtures-${number}`);
    }
  );
});
