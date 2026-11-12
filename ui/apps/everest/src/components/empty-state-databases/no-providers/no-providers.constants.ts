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

import type { SxProps, Theme } from '@mui/material';

// Suggested providers shown before the "browse the hub" tile. Kept small so the
// empty state previews the provider grid without turning into a catalog.
export const SPOTLIGHT_LIMIT = 2;

// Hub extension names suggested first, in this order. Curated here until the
// hub publishes a featured/popularity signal; missing entries are skipped and
// their slots filled by ranking the rest of the catalog.
export const SPOTLIGHT_PREFERRED_PROVIDERS = ['percona-postgresql', 'valkey'];

export const EXTENSION_HUB_WEBSITE_URL = 'https://openeverest.io/extensions/';

// Sub-paths handed to the plugin-hub UI through the plugin route
// (`/plugins/plugin-hub/*`), so it can focus the provider list or a provider.
export const HUB_PROVIDERS_SUB_PATH = 'providers';

export const browseTileContentSx: SxProps<Theme> = {
  height: '100%',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  gap: 1,
  p: 2,
  textAlign: 'center',
};

// Text followed by a trailing icon, e.g. "How to install →".
export const inlineIconLabelSx: SxProps<Theme> = {
  display: 'flex',
  alignItems: 'center',
  gap: 0.5,
};
