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

import { buildCreateInstanceSpec } from 'hooks/api/db-instances/useCreateDbInstance';
import {
  PRESET_NAME,
  PRESET_PROVIDER,
  TEAM_A,
  TEAM_B,
  clusterPreset,
  clusterPresetItem,
  clusterPresetSpec,
  teamAPreset,
  teamAPresetItem,
  teamAPresetSpec,
  teamBPresetItem,
} from './__mocks__/presets';
import { PresetScope, SelectablePreset } from './preset-selection.types';
import {
  buildPresetCreateArgs,
  getClusterPresetId,
  getNamespacePresetId,
  getPresetBackupFormValues,
  mergePresets,
  parsePresetId,
} from './preset-selection.utils';

const createArgs = (selectedPreset: SelectablePreset) =>
  buildPresetCreateArgs({
    selectedPreset,
    provider: PRESET_PROVIDER,
    dbName: 'my-db',
    k8sNamespace: TEAM_A,
  });

describe('buildPresetCreateArgs', () => {
  it('sends the selected preset spec verbatim (same reference)', () => {
    expect(createArgs(clusterPreset).formValue.spec).toBe(clusterPreset.spec);
  });

  // The server looks the preset up by the kind the annotation names, and
  // refuses an instance carrying both: exactly one, matching the scope.
  it('records a cluster preset in the cluster annotation only', () => {
    expect(createArgs(clusterPreset).annotations).toEqual({
      'openeverest.io/cluster-instance-preset': PRESET_NAME,
    });
  });

  it('records a namespace preset in the namespace annotation only', () => {
    expect(createArgs(teamAPreset).annotations).toEqual({
      'openeverest.io/instance-preset': PRESET_NAME,
    });
  });

  it('produces a spec of providerRef + preset spec, with no wizard meta leaking', () => {
    const instanceSpec = buildCreateInstanceSpec({
      ...createArgs(teamAPreset).formValue,
      presetId: teamAPreset.id,
    });

    expect(instanceSpec).toEqual({
      ...teamAPresetSpec,
      providerRef: { name: PRESET_PROVIDER },
    });
    // Request-address / UI-only fields must never end up in the Instance spec.
    expect(instanceSpec).not.toHaveProperty('dbName');
    expect(instanceSpec).not.toHaveProperty('k8sNamespace');
    expect(instanceSpec).not.toHaveProperty('presetId');
    expect(instanceSpec).not.toHaveProperty('provider');
  });
});

describe('mergePresets', () => {
  it('gives same-named cluster and namespace presets distinct ids', () => {
    const [first, second] = mergePresets(
      [clusterPresetItem],
      [teamAPresetItem],
      TEAM_A
    );

    expect(first.scope).toBe(PresetScope.Cluster);
    expect(second.scope).toBe(PresetScope.Namespace);
    expect(first.id).not.toBe(second.id);
    expect(first.name).toBe(second.name);
  });

  it('gives a same-named preset a different id in another namespace', () => {
    const [inTeamA] = mergePresets([], [teamAPresetItem], TEAM_A);
    const [inTeamB] = mergePresets([], [teamBPresetItem], TEAM_B);

    expect(inTeamA.id).not.toBe(inTeamB.id);
  });

  it('skips items that cannot be referenced by name', () => {
    expect(
      mergePresets(
        [{ spec: clusterPresetSpec }],
        [{ metadata: {}, spec: clusterPresetSpec }],
        TEAM_A
      )
    ).toEqual([]);
  });
});

describe('parsePresetId', () => {
  it('reads back what the id builders wrote', () => {
    expect(parsePresetId(getClusterPresetId(PRESET_NAME))).toEqual({
      scope: PresetScope.Cluster,
      name: PRESET_NAME,
    });
    expect(parsePresetId(getNamespacePresetId(TEAM_A, PRESET_NAME))).toEqual({
      scope: PresetScope.Namespace,
      namespace: TEAM_A,
      name: PRESET_NAME,
    });
  });

  it('rejects an id it did not build', () => {
    expect(parsePresetId(PRESET_NAME)).toBeUndefined();
    expect(parsePresetId(`namespace/${PRESET_NAME}`)).toBeUndefined();
  });
});

// The preview reads these values while the request sends the preset spec, so
// they must describe what the preset really runs.
describe('getPresetBackupFormValues', () => {
  const storageRef = { name: 'bs-team-a' };
  const classRef = { name: 'psmdb-backup-class' };
  const daily = { name: 'daily', cron: '0 0 * * *', enabled: true };

  it('lists the schedules and PITR of an enabled backup', () => {
    expect(getPresetBackupFormValues(teamAPresetSpec)).toEqual({
      classRef,
      schedules: [{ ...daily, storageName: 'bs-team-a' }],
      pitr: { 'bs-team-a': { enabled: true } },
    });
  });

  it('lists nothing when the preset switches backups off', () => {
    const { schedules, pitr } = getPresetBackupFormValues({
      ...clusterPresetSpec,
      backup: {
        enabled: false,
        classRef,
        storages: [{ storageRef, schedules: [daily], pitr: { enabled: true } }],
      },
    });

    expect(schedules).toEqual([]);
    expect(pitr).toEqual({});
  });

  it('leaves out a schedule the preset disables', () => {
    const { schedules } = getPresetBackupFormValues({
      ...clusterPresetSpec,
      backup: {
        enabled: true,
        classRef,
        storages: [
          {
            storageRef,
            schedules: [daily, { ...daily, name: 'weekly', enabled: false }],
          },
        ],
      },
    });

    expect(schedules.map(({ name }) => name)).toEqual(['daily']);
  });

  it('keeps PITR that runs without any schedule', () => {
    const { schedules, pitr } = getPresetBackupFormValues({
      ...clusterPresetSpec,
      backup: {
        enabled: true,
        classRef,
        storages: [{ storageRef, pitr: { enabled: true } }],
      },
    });

    expect(schedules).toEqual([]);
    expect(pitr).toEqual({ 'bs-team-a': { enabled: true } });
  });

  it('lists nothing for a preset without backups', () => {
    const { schedules, pitr } = getPresetBackupFormValues(clusterPresetSpec);

    expect(schedules).toEqual([]);
    expect(pitr).toEqual({});
  });
});
