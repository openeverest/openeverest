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

// Test fixture: a plugin bundle the host loader imports via descriptor.bundleUrl.
// It registers one of each of a sidebarItem, a route, and a globalDashboardWidget
// so the loader's extension-type gating can be exercised, and it calls the
// host-provided fetch so the proxy prefixing/auth headers can be asserted.
import type { PluginRegisterFn } from '@openeverest/plugin-sdk';

const NoopComponent = () => null;

export const register: PluginRegisterFn = (api) => {
  api.registerExtension({ type: 'sidebarItem', label: 'Hub' });
  api.registerExtension({
    type: 'route',
    label: 'Hub',
    component: NoopComponent,
  });
  api.registerExtension({
    type: 'globalDashboardWidget',
    label: 'Widget',
    component: NoopComponent,
  });

  void api.fetch('/context', { headers: { 'X-Test': '1' } });
};
