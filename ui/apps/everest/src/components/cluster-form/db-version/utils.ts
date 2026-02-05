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

import { gt, gte, coerce } from 'semver';
import { DbEngine, DbEngineType } from 'shared-types/dbEngines.types';
import { WizardMode } from 'shared-types/wizard.types';

export const filterAvailableDbVersionsForDbEngineEdition = (
  dbEngine: DbEngine,
  currentVersion: string,
  mode: WizardMode
) => {
  let versions = dbEngine.availableVersions.engine || [];
  const currentSemverVersion = coerce(currentVersion);
  const dbType = dbEngine.type;

  if (!currentSemverVersion) {
    return versions;
  }

  const currentMajor = currentSemverVersion.major;
  const currentMinor = currentSemverVersion.minor;

  // Filter out downgrades
  versions = versions.filter(({ version }) => {
    const semverVersion = coerce(version);
    const checkVersion =
      mode === WizardMode.Restore
        ? gte(semverVersion!, currentSemverVersion)
        : gt(semverVersion!, currentSemverVersion);
    return semverVersion ? checkVersion : true;
  });

  // If the engine is PXC or PG, major version upgrades are also ruled out
  if ([DbEngineType.PXC, DbEngineType.POSTGRESQL].includes(dbType)) {
    versions = versions.filter(({ version }) => {
      const semverVersion = coerce(version);
      return semverVersion ? semverVersion.major === currentMajor : true;
    });
  }

  // Rule out skipping major versions
  versions = versions.filter(({ version }) => {
    const semverVersion = coerce(version);
    return semverVersion ? semverVersion.major - currentMajor <= 1 : true;
  });

  // Disallow upgrading from PXC 8.0.x to 8.4.x
  if (dbType === DbEngineType.PXC) {
    versions = versions.filter(({ version }) => {
      const semverVersion = coerce(version);
      return !(
        currentMajor === 8 &&
        currentMinor === 0 &&
        semverVersion?.major === 8 &&
        semverVersion?.minor === 4
      );
    });
  }

  return versions;
};
