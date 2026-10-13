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

import { render, screen, waitFor } from '@testing-library/react';
import { userEvent } from 'vitest/browser';
import { TestWrapper } from 'utils/test';
import { Instance } from 'shared-types/api.types';
import {
  SchemaSectionCard,
  useClusterOverviewData,
} from './hooks/use-cluster-overview-data';
import { ClusterOverview } from './cluster-overview';

vi.mock('./hooks/use-cluster-overview-data', () => ({
  useClusterOverviewData: vi.fn(),
}));

const instance: Instance = {
  apiVersion: 'core.openeverest.io/v1alpha1',
  kind: 'Instance',
  metadata: {},
  spec: { providerRef: { name: 'test-provider' } },
  status: { phase: 'Ready' },
};

const card = (key: string, title: string, rows: number): SchemaSectionCard => ({
  key,
  title,
  fields: Array.from({ length: rows }, (_, index) => ({
    label: `Field ${index + 1}`,
    path: `spec.${key}.${index}`,
    value: 'value',
  })),
});

const providerCards = (resourcesRows = 4) => [
  card('version', 'Version', 1),
  card('resources', 'Resources', resourcesRows),
  card('monitoring', 'Monitoring', 1),
  card('advanced', 'Advanced configuration', 3),
  card('scheduling', 'Pod scheduling policy', 2),
];

const mockOverview = (cards: SchemaSectionCard[]) =>
  vi.mocked(useClusterOverviewData).mockReturnValue({
    instanceName: 'db',
    namespace: 'ns',
    instance,
    isLoading: false,
    credentials: undefined,
    schemaSectionCards: cards,
    otherFields: [],
    provider: undefined,
    sections: {},
    backupsSupported: false,
    selectedTopology: undefined,
  });

const overview = (width: number) => (
  <TestWrapper>
    <div style={{ width }}>
      <ClusterOverview />
    </div>
  </TestWrapper>
);

const CARD_IDS = [
  'database-details',
  'version-details',
  'resources-details',
  'monitoring-details',
  'advanced-details',
  'scheduling-details',
];

// Relative to the overview, so scrolling the test page doesn't shift them.
const rectsById = () => {
  const origin = screen.getByTestId('cluster-overview').getBoundingClientRect();
  return Object.fromEntries(
    CARD_IDS.map((id) => {
      const rect = screen.getByTestId(id).getBoundingClientRect();
      return [
        id,
        new DOMRect(
          rect.left - origin.left,
          rect.top - origin.top,
          rect.width,
          rect.height
        ),
      ];
    })
  );
};

const columnCount = () =>
  new Set(Object.values(rectsById()).map((rect) => Math.round(rect.left))).size;

const hasOverlaps = () => {
  const rects = Object.values(rectsById());
  return rects.some((a, index) =>
    rects
      .slice(index + 1)
      .some(
        (b) =>
          a.left < b.right - 1 &&
          b.left < a.right - 1 &&
          a.top < b.bottom - 1 &&
          b.top < a.bottom - 1
      )
  );
};

describe('ClusterOverview layout', () => {
  it.each([
    [800, 1],
    [1200, 2],
    [1500, 3],
  ])(
    'lays a %ipx overview out in %i columns without overlaps',
    async (width, columns) => {
      mockOverview(providerCards());

      render(overview(width));

      await waitFor(() => expect(columnCount()).toBe(columns));
      expect(hasOverlaps()).toBe(false);
    }
  );

  it('re-flows into more columns when the overview widens', async () => {
    const errors: string[] = [];
    const onError = (event: ErrorEvent) => errors.push(event.message);
    window.addEventListener('error', onError);
    mockOverview(providerCards());
    const { rerender } = render(overview(1200));
    await waitFor(() => expect(columnCount()).toBe(2));

    rerender(overview(1500));

    await waitFor(() => expect(columnCount()).toBe(3));
    expect(hasOverlaps()).toBe(false);
    window.removeEventListener('error', onError);
    // Laying out inside a ResizeObserver callback would report a loop error.
    expect(errors).toEqual([]);
  });

  it('keeps every card in its column when one grows after the user interacts', async () => {
    mockOverview(providerCards());
    const { rerender } = render(overview(1200));
    await waitFor(() => expect(columnCount()).toBe(2));
    const before = rectsById();

    await userEvent.click(screen.getByText('Monitoring'));
    mockOverview(providerCards(15));
    rerender(overview(1200));

    await waitFor(() =>
      expect(rectsById()['resources-details'].height).toBeGreaterThan(
        before['resources-details'].height
      )
    );
    const after = rectsById();
    const growth =
      after['resources-details'].height - before['resources-details'].height;
    const resourcesLeft = before['resources-details'].left;
    const resourcesTop = before['resources-details'].top;

    CARD_IDS.forEach((id) => {
      expect(after[id].left).toBe(before[id].left);
      const pushedDown =
        before[id].left === resourcesLeft && before[id].top > resourcesTop;
      expect(after[id].top).toBeCloseTo(
        before[id].top + (pushedDown ? growth : 0),
        0
      );
    });
    expect(hasOverlaps()).toBe(false);
  });
});
