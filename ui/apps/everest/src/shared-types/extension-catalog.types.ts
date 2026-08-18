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

// Types describing the OpenEverest Extension Hub index (`hub.openeverest.io/v1`).
// These mirror the JSON returned by the plugin-hub plugin at
// `/v1/plugins/plugin-hub/api/summary`. They are hand-written because the hub
// schema is not part of the OpenEverest CRD-generated types.

export type ExtensionType = 'provider' | 'plugin';

export type ExtensionMaturity = 'alpha' | 'beta' | 'stable' | 'deprecated';

export interface ExtensionChannel {
  ref?: string;
  version?: string;
  digest?: string;
}

export interface ExtensionArtifact {
  defaultChannel?: string;
  channels?: Record<string, ExtensionChannel>;
}

export interface ExtensionProviderInfo {
  providerName: string;
  supportedEngines?: string[];
}

export interface ExtensionPluginInfo {
  contributes?: {
    ui?: boolean;
    cli?: boolean;
    backend?: boolean;
  };
  extensionPoints?: string[];
}

export interface ExtensionCatalogEntry {
  name: string;
  type: ExtensionType;
  displayName?: string;
  description?: string;
  icon?: string;
  homepage?: string;
  sourceRepo?: string;
  license?: string;
  verified?: boolean;
  health?: string;
  maturity?: ExtensionMaturity;
  categories?: string[];
  keywords?: string[];
  artifacts?: {
    chart?: ExtensionArtifact;
    frontend?: ExtensionArtifact;
  };
  provider?: ExtensionProviderInfo | null;
  plugin?: ExtensionPluginInfo | null;
}

export interface ExtensionIndex {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    catalogId?: string;
    generatedAt?: string;
    schemaVersion?: string;
    totalExtensions?: number;
  };
  extensions?: ExtensionCatalogEntry[];
}

// Resolved, UI-friendly metadata for a single extension.
export interface ResolvedExtensionMeta {
  name: string;
  displayName: string;
  description?: string;
  icon?: string;
  version?: string;
  categories?: string[];
  maturity?: ExtensionMaturity;
}

// Untrusted shapes for data coming off the wire. The hub is an independent
// project that may drift from our hand-written `ExtensionCatalogEntry` /
// `ExtensionIndex` types, so callers must guard every access. Use
// `normalizeEntry` in the extension-catalog hook as the single narrowing point;
// downstream code can then trust `ResolvedExtensionMeta`.
export type RawExtensionCatalogEntry = Partial<ExtensionCatalogEntry>;

export interface RawExtensionIndex {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    catalogId?: string;
    generatedAt?: string;
    schemaVersion?: string;
    totalExtensions?: number;
  };
  extensions?: RawExtensionCatalogEntry[];
}
