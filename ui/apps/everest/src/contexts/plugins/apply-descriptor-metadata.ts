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

import type { Extension } from '@openeverest/plugin-sdk';

// Shape of a single extension-point entry as delivered by GET /v1/plugins,
// mirroring spec.frontend.extensionPoints[] on the Plugin CR.
export interface ExtensionPointDescriptor {
  type: string;
  label?: string;
  path?: string;
  icon?: string;
  providers?: string[];
}

// Reconcile a bundle-registered extension with the matching entry in the
// Plugin CR. Tabs and settings panels are matched by type+path; the rest by
// type+label. The CR is the source of truth for provider filtering, so a
// declared `providers` list overrides whatever the bundle registered — this
// lets operators scope a plugin without rebuilding it.
export function applyDescriptorMetadata(
  ext: Extension,
  extensionPoints: ExtensionPointDescriptor[]
): void {
  const extPath = 'path' in ext ? ext.path : undefined;
  const match = extensionPoints.find((ep) => {
    if (ep.type !== ext.type) {
      return false;
    }
    if (extPath && ep.path) {
      return ep.path === extPath;
    }
    return ep.label === ext.label;
  });
  if (!match) {
    return;
  }

  if (
    match.providers?.length &&
    (ext.type === 'clusterDetailTab' ||
      ext.type === 'clusterAction' ||
      ext.type === 'clusterCard' ||
      ext.type === 'instanceCreateFormSection' ||
      ext.type === 'instanceEditFormSection')
  ) {
    ext.providers = match.providers;
  }

  // The bundle may omit the sidebar icon; the backend already resolves the
  // CR's relative path to a full proxy URL.
  if (ext.type === 'sidebarItem' && !ext.icon && match.icon) {
    ext.icon = match.icon;
  }
}
