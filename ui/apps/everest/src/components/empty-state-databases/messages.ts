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
  createFirstResource: 'Create your first resource',
  pickProvider: 'Pick a provider to get started.',
  noPermissions: 'You lack permission to create a resource.',
  noProvidersBody:
    'No providers are installed yet. Install one to create databases, storage or AI services.',
  howToInstall: 'How to install',
  browseHub: 'Browse Extension Hub',
  moreProviders: (count: number) =>
    `+${count} more ${count === 1 ? 'provider' : 'providers'}`,
  autoRefreshHint:
    'This page updates automatically once a provider is installed.',
};
