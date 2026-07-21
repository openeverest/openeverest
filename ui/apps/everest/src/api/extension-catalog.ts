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

import { api } from './api';
import { RawExtensionIndex } from 'shared-types/extension-catalog.types';

// Name of the plugin-hub Plugin CR. The hub install convention uses the
// release name "plugin-hub" (see hub formula install.helm.releaseName).
export const PLUGIN_HUB_NAME = 'plugin-hub';

// Fetches the extension catalog summary served by the plugin-hub plugin.
// The request is proxied through the OpenEverest backend
// (`/v1/plugins/plugin-hub/api/summary`). Returns null when the plugin-hub
// plugin is not installed or the catalog cannot be reached, so callers can
// gracefully fall back to raw resource names. The return type is deliberately
// `RawExtensionIndex` (a Partial-shaped mirror) — the hub is an external
// project and its schema may drift, so consumers must guard, not trust.
export const getExtensionCatalogFn =
  async (): Promise<RawExtensionIndex | null> => {
    try {
      const response = await api.get<RawExtensionIndex>(
        `/plugins/${PLUGIN_HUB_NAME}/api/summary`,
        { disableNotifications: true }
      );
      return response.data ?? null;
    } catch {
      return null;
    }
  };
