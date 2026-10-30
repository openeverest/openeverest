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

import {
  Stack,
  ThemeProvider,
  Typography,
  alpha,
  createTheme,
} from '@mui/material';
import { everestThemeOptions } from '@percona/design';
import { EverestHorizontalIcon } from '@percona/ui-lib';
import { Messages } from 'pages/login/Login.messages';
import { LoginLinks } from 'pages/login/login-links/login-links';

// The brand panel is always dark, regardless of the user's color mode.
const brandPanelTheme = createTheme(everestThemeOptions('dark'));

export const LoginBrandPanel = () => (
  <ThemeProvider theme={brandPanelTheme}>
    <Stack
      data-testid="login-brand-panel"
      sx={(theme) => ({
        flex: '0 0 42%',
        justifyContent: 'space-between',
        gap: 6,
        p: { md: 5, lg: 8 },
        color: theme.palette.text.primary,
        background: [
          `radial-gradient(circle at 0% 0%, ${alpha(theme.palette.primary.dark, 0.28)}, transparent 55%)`,
          `radial-gradient(circle at 100% 100%, ${alpha(theme.palette.info.dark, 0.18)}, transparent 50%)`,
          theme.palette.surfaces?.elevation0,
        ].join(', '),
      })}
    >
      <EverestHorizontalIcon
        sx={{ height: 56, width: 'auto', alignSelf: 'flex-start' }}
      />
      <Typography variant="h4" component="p" sx={{ maxWidth: 480 }}>
        {Messages.pitch}
      </Typography>
      <LoginLinks />
    </Stack>
  </ThemeProvider>
);
