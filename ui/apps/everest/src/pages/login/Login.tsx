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

import { useContext, useEffect } from 'react';
import { Stack } from '@mui/material';
import { enqueueSnackbar } from 'notistack';
import { Navigate } from 'react-router-dom';
import { EverestHorizontalIcon } from '@percona/ui-lib';
import { AuthContext } from 'contexts/auth';
import { SSO_LOGIN_ERROR_KEY } from 'contexts/auth/auth.utils';
import { useActiveBreakpoint } from 'hooks/utils/useActiveBreakpoint';
import { LoginBrandPanel } from './login-brand-panel/login-brand-panel';
import { LoginForm } from './login-form/login-form';
import { LoginLinks } from './login-links/login-links';
import { Messages } from './Login.messages';

const Login = () => {
  const { authStatus, redirectRoute } = useContext(AuthContext);
  // Below md the split-screen panel squeezes the form, so we switch to a
  // single centered column instead
  const { isBelowMd: isSingleColumn } = useActiveBreakpoint();

  useEffect(() => {
    // Surface an SSO sign-in failure that redirected back here from
    // login-callback
    if (sessionStorage.getItem(SSO_LOGIN_ERROR_KEY)) {
      sessionStorage.removeItem(SSO_LOGIN_ERROR_KEY);
      enqueueSnackbar(Messages.ssoLoginError, { variant: 'error' });
    }
  }, []);

  if (authStatus === 'unknown') {
    return <></>;
  }

  if (authStatus === 'loggedIn') {
    return <Navigate to={redirectRoute ?? '/'} replace />;
  }

  return (
    <Stack direction="row" sx={{ minHeight: '100dvh' }}>
      {!isSingleColumn && <LoginBrandPanel />}
      <Stack
        sx={{
          flex: 1,
          alignItems: 'center',
          justifyContent: 'center',
          px: 3,
          pt: 6,
          // In single-column mode, bias the content slightly above the optical
          // center
          pb: { xs: '12dvh', md: 6 },
          bgcolor: 'background.default',
        }}
      >
        <Stack sx={{ width: '100%', maxWidth: 360 }}>
          {isSingleColumn && (
            <EverestHorizontalIcon
              data-testid="login-single-column-logo"
              sx={{ height: 48, width: 'auto', alignSelf: 'center', mb: 5 }}
            />
          )}
          <LoginForm />
          {isSingleColumn && (
            <LoginLinks sx={{ justifyContent: 'center', mt: 5 }} />
          )}
        </Stack>
      </Stack>
    </Stack>
  );
};

export default Login;
