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

export const Messages = {
  source: (backup: string, storage: string) =>
    `Creating the database from backup ${backup} in storage ${storage}.`,
  versionMatched: (version: string) =>
    `Version ${version} is preselected because it matches the backup. This version is guaranteed to work.`,
  versionMissing: (engineVersion: string) =>
    `The backup was taken on version ${engineVersion}, which isn't available. Pick a version: compatibility is not guaranteed.`,
  versionUnknown: 'Pick the same database version as the backup.',
  topologyUnknown: 'Pick the same topology as the backup.',
  changeBackup: 'Change backup',
};
