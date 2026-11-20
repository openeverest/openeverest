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

import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import type { TileLinkIconProps } from './tile-link-icon.types';

// External links open in a new tab, so they get the "opens elsewhere" icon.
export const TileLinkIcon = ({ link }: TileLinkIconProps) =>
  'href' in link ? (
    <OpenInNewIcon fontSize="small" aria-hidden="true" />
  ) : (
    <ArrowForwardIcon fontSize="small" aria-hidden="true" />
  );
