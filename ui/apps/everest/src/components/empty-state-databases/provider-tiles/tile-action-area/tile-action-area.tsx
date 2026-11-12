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

import { CardActionArea } from '@mui/material';
import { Link } from 'react-router-dom';
import type { TileActionAreaProps } from './tile-action-area.types';

export const TileActionArea = ({
  link,
  testId,
  sx,
  children,
}: TileActionAreaProps) =>
  'href' in link ? (
    <CardActionArea
      data-testid={testId}
      href={link.href}
      target="_blank"
      rel="noopener noreferrer"
      sx={sx}
    >
      {children}
    </CardActionArea>
  ) : (
    <CardActionArea
      data-testid={testId}
      component={Link}
      to={link.to}
      state={link.state}
      sx={sx}
    >
      {children}
    </CardActionArea>
  );
