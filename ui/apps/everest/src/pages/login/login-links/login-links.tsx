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

import { Link, Stack, SxProps, Theme } from '@mui/material';
import { LOGIN_LINKS } from 'pages/login/Login.constants';

interface LoginLinksProps {
  sx?: SxProps<Theme>;
}

export const LoginLinks = ({ sx }: LoginLinksProps) => (
  <Stack
    component="nav"
    data-testid="login-links"
    direction="row"
    sx={[
      { flexWrap: 'wrap', columnGap: 3, rowGap: 1 },
      ...(Array.isArray(sx) ? sx : [sx]),
    ]}
  >
    {LOGIN_LINKS.map(({ label, href }) => (
      <Link
        key={href}
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        variant="body2"
        color="text.secondary"
        underline="hover"
      >
        {label}
      </Link>
    ))}
  </Stack>
);
