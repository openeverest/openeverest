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

import EmptyStateWithLink from 'components/empty-state-with-link';
import { SettingsTabs } from 'pages/settings/settings.types';
import { Messages } from '../backups.messages';

/**
 * Shown on the Backups tab when the namespace has no backup storages configured.
 *
 * The card stays observation-only ("Overview = observe") — the CTA is a plain
 * router Link to the Storage Locations settings page, not a cluster-state
 * mutation.  This matches the schema-level component contract discussed in
 * issue #2185: card.cta: { kind: 'navigate', target: '/settings/storage-locations' }.
 */
export const NoStoragesMessage = () => {
  return (
    <EmptyStateWithLink
      message={Messages.noStoragesMessage}
      linkLabel={Messages.goToStorageSettings}
      to={`/settings/${SettingsTabs.storageLocations}`}
      dataTestId="no-storages-message"
    />
  );
};
