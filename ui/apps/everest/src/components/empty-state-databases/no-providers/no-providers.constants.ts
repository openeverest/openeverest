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

export const EXTENSION_HUB_WEBSITE_URL = 'https://openeverest.io/extensions/';

// Sub-paths handed to the plugin-hub UI through the plugin route
// (`/plugins/plugin-hub/*`), so it can focus the provider list or a provider.
export const HUB_PROVIDERS_SUB_PATH = 'providers';

export const browseTileContentSx: SxProps<Theme> = {
  height: '100%',
  minHeight: 160,
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  gap: 1,
  p: 2,
  textAlign: 'center',
};

export const browseTileActionSx: SxProps<Theme> = {
  display: 'flex',
  alignItems: 'center',
  gap: 0.5,
};

export const installHintSx: SxProps<Theme> = {
  display: 'flex',
  alignItems: 'center',
  gap: 0.5,
};
