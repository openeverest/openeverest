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

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  AuthProvider as OidcAuthProvider,
  AuthProviderProps as OidcAuthProviderProps,
  useAuth as useOidcAuth,
} from 'oidc-react';
import { AxiosError } from 'axios';
import { jwtDecode } from 'jwt-decode';
import {
  api,
  addApiErrorInterceptor,
  removeApiErrorInterceptor,
  addApiAuthInterceptor,
  removeApiAuthInterceptor,
} from 'api/api';
import { enqueueSnackbar } from 'notistack';
import { useQueryClient } from '@tanstack/react-query';
import AuthContext from './auth.context';
import { EVEREST_JWT_ISSUER } from 'consts';
import {
  AuthMode,
  AuthProviderProps,
  ManualAuthArgs,
  UserAuthStatus,
} from './auth.context.types';
import { isAfter } from 'date-fns';
import {
  initializeAuthorizerFetchLoop,
  stopAuthorizerFetchLoop,
} from 'utils/rbac';
import { useCrossTabAuth } from 'hooks/utils';

const Provider = ({
  oidcConfig,
  children,
}: {
  oidcConfig?: OidcAuthProviderProps;
  children: React.ReactNode;
}) => {
  const authProvider = useMemo(
    () => (
      <AuthProvider
        isSsoEnabled={!!oidcConfig?.authority && !!oidcConfig?.clientId}
      >
        {children}
      </AuthProvider>
    ),
    [children, oidcConfig]
  );
  return <OidcAuthProvider {...oidcConfig}>{authProvider}</OidcAuthProvider>;
};

const AuthProvider = ({ children, isSsoEnabled }: AuthProviderProps) => {
  const [authStatus, setAuthStatus] = useState<UserAuthStatus>('unknown');
  const [redirect, setRedirect] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const { signIn, userManager } = useOidcAuth();
  const checkAuth = useCallback(async (token: string) => {
    try {
      await api.get('/version', {
        headers: { Authorization: `Bearer ${token}` },
      });
      return true;
    } catch {
      return false;
    }
  }, []);

  const login = async (mode: AuthMode, manualAuthArgs?: ManualAuthArgs) => {
    setAuthStatus('loggingIn');
    if (mode === 'sso') {
      await signIn();
    } else {
      const { username, password } = manualAuthArgs!;
      try {
        const response = await api.post('/session', { username, password });
        const token = response.data.token; // Assuming the response structure has a token field
        localStorage.setItem('everestToken', token);
        setLoggedInStatus(username);
      } catch (error) {
        if (error instanceof AxiosError) {
          const errorStatus = error.response?.status;
          let errorMsg = 'Something went wrong';

          if (errorStatus === 401) {
            errorMsg = 'Invalid credentials';
          } else if (errorStatus === 429) {
            errorMsg =
              "Looks like you've made too many attempts. Try again later.";
          }
          enqueueSnackbar(errorMsg, {
            variant: 'error',
          });
        }
        setLogoutStatus();
        return;
      }
    }
  };

  const runLogoutCleanup = useCallback(async () => {
    setAuthStatus('loggedOut');
    localStorage.removeItem('everestToken');
    sessionStorage.clear();
    setRedirect(null);
    stopAuthorizerFetchLoop();
    queryClient.clear();
    removeApiErrorInterceptor();
    removeApiAuthInterceptor();
    if (isSsoEnabled) {
      await userManager.clearStaleState();
      await userManager.removeUser();
    }
  }, [isSsoEnabled, queryClient, userManager]);

  const { broadcastLogout } = useCrossTabAuth(runLogoutCleanup);

  const logout = async () => {
    const token = localStorage.getItem('everestToken');
    try {
      await api.delete('/session', { headers: { token: token } });
    } catch {
      // proceed with cleanup even if the session delete request fails
    }
    broadcastLogout();
    await runLogoutCleanup();
  };

  const setRedirectRoute = (route: string) => {
    setRedirect(route);
  };

  const setLoggedInStatus = (username: string) => {
    setAuthStatus('loggedIn');
    addApiErrorInterceptor();
    addApiAuthInterceptor();
    initializeAuthorizerFetchLoop(username);
  };

  const setLogoutStatus = useCallback(async () => {
    setAuthStatus('loggedOut');
    localStorage.removeItem('everestToken');
    queryClient.clear();
    if (isSsoEnabled) {
      await userManager.clearStaleState();
      await userManager.removeUser();
    }
    stopAuthorizerFetchLoop();
  }, [isSsoEnabled, queryClient, userManager]);

  const silentlyRenewToken = useCallback(async () => {
    try {
      const newLoggedUser = await userManager.signinSilent();
      if (newLoggedUser && newLoggedUser.access_token) {
        localStorage.setItem('everestToken', newLoggedUser.access_token);
      } else {
        setLogoutStatus();
      }
    } catch (error) {
      setLogoutStatus();
    }
  }, [userManager]);

  useEffect(() => {
    if (isSsoEnabled) {
      userManager.events.addUserLoaded((user) => {
        localStorage.setItem('everestToken', user.access_token || '');
        const decoded = jwtDecode(user.access_token || '');
        setLoggedInStatus(decoded.sub || '');
      });

      userManager.events.addAccessTokenExpiring(() => {
        silentlyRenewToken();
      });

      userManager.signinSilentCallback();
    }
  }, [isSsoEnabled, silentlyRenewToken, userManager]);

  useEffect(() => {
    if (window.location !== window.parent.location) {
      // This is running in the iframe, so we are renewing the token silently
      return;
    }

    if (authStatus === 'loggedIn' || authStatus === 'loggingIn') {
      return;
    }

    const authRoutine = async (token: string) => {
      try {
        const decoded = jwtDecode(token);
        const iss = decoded.iss;
        const exp = decoded.exp;
        if (iss === EVEREST_JWT_ISSUER) {
          const isTokenValid = await checkAuth(token);
          const username =
            decoded.sub?.substring(0, decoded.sub.indexOf(':')) || '';
          if (isTokenValid) {
            setLoggedInStatus(username);
          } else {
            setLogoutStatus();
          }
        } else {
          if (isAfter(new Date(), new Date((exp || 0) * 1000))) {
            silentlyRenewToken();
            return;
          }

          const user = await userManager.getUser();

          if (!user) {
            setLogoutStatus();
          } else {
            setLoggedInStatus(decoded.sub || '');
            return;
          }
        }
      } catch (error) {
        logout();
      }
    };
    const savedToken = localStorage.getItem('everestToken');

    if (!savedToken) {
      setLogoutStatus();
      return;
    }

    authRoutine(savedToken);
  }, [authStatus, silentlyRenewToken, userManager]);

  return (
    <AuthContext.Provider
      value={{
        login,
        logout,
        authStatus,
        redirectRoute: redirect,
        setRedirectRoute,
        isSsoEnabled,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export default Provider;
