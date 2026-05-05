// everest
// Copyright (C) 2023 Percona LLC
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

import { DbEngine, DbEngineType } from 'shared-types/dbEngines.types';
import { filterAvailableDbVersionsForDbEngineEdition } from './utils';
import { WizardMode } from 'shared-types/wizard.types';

const generateDbEngineWithVersions = (
  versions: string[],
  type: DbEngineType
): DbEngine =>
  ({
    availableVersions: {
      engine: versions.map((version) => ({ version })),
    },
    type,
  }) as unknown as DbEngine;

describe('DBVersion Available filter test', () => {
  describe('filterAvailableDbVersionsForDbEngineEdition', () => {
    it('should rule out downgrades', () => {
      expect(
        filterAvailableDbVersionsForDbEngineEdition(
          generateDbEngineWithVersions(
            [
              '8.0.40',
              '8.0.39-30.1',
              '8.0.36-28.1',
              '8.0.35-27.1',
              '5.7.44-31.65',
            ],
            DbEngineType.PXC
          ),
          '8.0.39-30.1',
          WizardMode.Edit
        ).map(({ version }) => version)
      ).toEqual(['8.0.40']);
      expect(
        filterAvailableDbVersionsForDbEngineEdition(
          generateDbEngineWithVersions(
            ['7.0.15-9', '7.0.14-8', '7.0.12-7', '6.0.19-16'],
            DbEngineType.PSMDB
          ),
          '7.0.15-9',
          WizardMode.Edit
        ).map(({ version }) => version)
      ).toEqual([]);
      expect(
        filterAvailableDbVersionsForDbEngineEdition(
          generateDbEngineWithVersions(
            ['16.4', '16.3', '15.8', '16.6'],
            DbEngineType.POSTGRESQL
          ),
          '16.4',
          WizardMode.Edit
        ).map(({ version }) => version)
      ).toEqual(['16.6']);
    });

    it('should not include own version', () => {
      expect(
        filterAvailableDbVersionsForDbEngineEdition(
          generateDbEngineWithVersions(
            ['13.0', '14.0', '14.1', '15.0'],
            DbEngineType.POSTGRESQL
          ),
          '14.1',
          WizardMode.Edit
        ).map(({ version }) => version)
      ).toEqual([]);
    });

    it('should allow major upgrade to the next version for PSMDB', () => {
      expect(
        filterAvailableDbVersionsForDbEngineEdition(
          generateDbEngineWithVersions(
            ['8.0.4-1', '7.0.15-9', '7.0.14-8', '6.0.19-16', '6.0.18-15'],
            DbEngineType.PSMDB
          ),
          '6.0.18-15',
          WizardMode.Edit
        ).map(({ version }) => version)
      ).toEqual(['7.0.15-9', '7.0.14-8', '6.0.19-16']);
    });

    it('should rule out major upgrades for PXC/PG', () => {
      expect(
        filterAvailableDbVersionsForDbEngineEdition(
          generateDbEngineWithVersions(
            ['9.0.0', '8.3.2-2.1', '8.0.39-30.1', '8.0.36-28.1', '8.0.35-27.1'],
            DbEngineType.PXC
          ),
          '8.0.36-28.1',
          WizardMode.Edit
        ).map(({ version }) => version)
      ).toEqual(['8.3.2-2.1', '8.0.39-30.1']);
      expect(
        filterAvailableDbVersionsForDbEngineEdition(
          generateDbEngineWithVersions(
            ['16.4', '16.3', '15.8', '15.7'],
            DbEngineType.POSTGRESQL
          ),
          '15.7',
          WizardMode.Edit
        ).map(({ version }) => version)
      ).toEqual(['15.8']);
    });
  });
});
