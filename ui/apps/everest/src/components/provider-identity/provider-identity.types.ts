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

import type { TypographyProps } from '@mui/material';
import type { Provider } from 'shared-types/api.types';
import type { ResolvedExtensionMeta } from 'shared-types/extension-catalog.types';

export interface ProviderIdentityProps {
  label: string;
  meta?: ResolvedExtensionMeta;
  // When true, render the description helper (tooltip icon) if meta.description
  // is available. Used by the tiles grid; drawer rows leave this off.
  showDescription?: boolean;
  // When true, clamp the label to a single ellipsised line. Used by drawer rows.
  ellipsis?: boolean;
  labelVariant?: TypographyProps['variant'];
}

export interface ProviderDisplay {
  name: string;
  label: string;
  meta?: ResolvedExtensionMeta;
}

export type GetProviderMeta = (
  providerName?: string | null
) => ResolvedExtensionMeta | undefined;

export type ResolveProviderDisplayInput = Pick<Provider, 'metadata'>;
