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
import {
  setAccessToken,
  getAccessToken,
  getAccessTokenExpiry,
  clearAccessToken,
  refreshSession,
} from 'api/session-token';
import { enqueueSnackbar } from 'notistack';
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
import { components } from '../../../../../api/http-api.types';

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

  const { signIn, userManager } = useOidcAuth();

  const login = async (mode: AuthMode, manualAuthArgs?: ManualAuthArgs) => {
    setAuthStatus('loggingIn');
    if (mode === 'sso') {
      await signIn();
    } else {
      const { username, password } = manualAuthArgs!;
      try {
        const response = await api.post<
          components['schemas']['AuthTokenResponse']
        >('/auth/token', {
          grant_type: 'password',
          username,
          password,
          // The refresh token is delivered as an HttpOnly cookie, never
          // exposed to JS. The access token is kept in memory only.
          refresh_token_delivery: 'cookie',
        } satisfies components['schemas']['AuthTokenRequest']);
        setAccessToken(response.data.access_token, response.data.expires_in);
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

  const logout = async () => {
    try {
      // Revokes the refresh token (carried by the HttpOnly cookie) and
      // blocklists the current access JWT.
      await api.post('/auth/revoke', {});
    } catch {
      // Best-effort: local session state is cleared regardless.
    }
    if (isSsoEnabled) {
      await userManager.clearStaleState();
      await setLogoutStatus();
    }

    setAuthStatus('loggedOut');
    clearAccessToken();
    localStorage.removeItem('everestToken');
    sessionStorage.clear();
    setRedirect(null);
    removeApiErrorInterceptor();
    removeApiAuthInterceptor();
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
    clearAccessToken();
    localStorage.removeItem('everestToken');
    if (isSsoEnabled) {
      await userManager.clearStaleState();
      await userManager.removeUser();
    }
    stopAuthorizerFetchLoop();
  }, [userManager]);

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
        const token = user.access_token;
        if (!token) {
          return;
        }
        localStorage.setItem('everestToken', token);
        const decoded = jwtDecode(token);
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

    // OIDC sessions are persisted in localStorage by oidc-react.
    const oidcAuthRoutine = async (token: string) => {
      try {
        const decoded = jwtDecode(token);
        const exp = decoded.exp;
        if (isAfter(new Date(), new Date((exp || 0) * 1000))) {
          silentlyRenewToken();
          return;
        }

        const user = await userManager.getUser();

        if (!user) {
          setLogoutStatus();
        } else {
          setLoggedInStatus(decoded.sub || '');
        }
      } catch (error) {
        logout();
      }
    };

    const bootstrapSession = async () => {
      // Try to restore an internal session: the in-memory access token is
      // lost on reload, but the HttpOnly refresh token cookie (if present)
      // can be exchanged for a fresh token pair.
      const accessToken = getAccessToken() || (await refreshSession());
      if (accessToken) {
        try {
          const decoded = jwtDecode(accessToken);
          if (decoded.iss === EVEREST_JWT_ISSUER) {
            const username =
              decoded.sub?.substring(0, decoded.sub.indexOf(':')) || '';
            setLoggedInStatus(username);
            return;
          }
        } catch {
          clearAccessToken();
        }
      }

      const savedToken = localStorage.getItem('everestToken');

      if (!savedToken) {
        setLogoutStatus();
        return;
      }

      oidcAuthRoutine(savedToken);
    };

    bootstrapSession();
  }, [authStatus, silentlyRenewToken, userManager]);

  // Proactively rotates internal sessions shortly before the access token
  // expires, so requests rarely hit a 401.
  useEffect(() => {
    if (authStatus !== 'loggedIn' || !getAccessToken()) {
      return;
    }

    let timer: ReturnType<typeof setTimeout>;
    const schedule = () => {
      const expiry = getAccessTokenExpiry();
      if (!expiry) {
        return;
      }
      const delay = Math.max(expiry - Date.now() - 60 * 1000, 5 * 1000);
      timer = setTimeout(async () => {
        const token = await refreshSession();
        if (token) {
          schedule();
        } else {
          enqueueSnackbar(
            'Your session has expired. Please sign in again.',
            { variant: 'info' }
          );
          setLogoutStatus();
        }
      }, delay);
    };
    schedule();

    return () => clearTimeout(timer);
  }, [authStatus, setLogoutStatus]);

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
