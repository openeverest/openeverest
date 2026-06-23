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

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
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
import type { HttpApi } from '@generated/api-types';

const LOGOUT_SYNC_CHANNEL = 'everest-auth-sync';
const LOGOUT_SYNC_STORAGE_KEY = 'everest-auth-sync';

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
  const logoutSyncRef = useRef<BroadcastChannel | null>(null);
  const tabIdRef = useRef(`tab-${Math.random().toString(36).slice(2)}`);

  const { signIn, userManager } = useOidcAuth();

  const login = async (mode: AuthMode, manualAuthArgs?: ManualAuthArgs) => {
    setAuthStatus('loggingIn');
    if (mode === 'sso') {
      await signIn();
    } else {
      const { username, password } = manualAuthArgs!;
      try {
        const response = await api.post<
          HttpApi.components['schemas']['AuthTokenResponse']
        >('/auth/token', {
          grant_type: 'password',
          username,
          password,
          // The refresh token is delivered as an HttpOnly cookie, never
          // exposed to JS. The access token is kept in memory only.
          refresh_token_delivery: 'cookie',
        } satisfies HttpApi.components['schemas']['AuthTokenRequest']);
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

  const broadcastLogoutSync = useCallback(() => {
    const payload = {
      type: 'logout',
      sender: tabIdRef.current,
      ts: Date.now(),
    };

    if (logoutSyncRef.current) {
      logoutSyncRef.current.postMessage(payload);
      return;
    }

    // Fallback for environments without BroadcastChannel support.
    localStorage.setItem(LOGOUT_SYNC_STORAGE_KEY, JSON.stringify(payload));
  }, []);

  const logout = async () => {
    try {
      // Revokes the refresh token (carried by the HttpOnly cookie) and
      // blocklists the current access JWT.
      await api.post('/auth/revoke', {});
    } catch {
      // Best-effort: local session state is cleared regardless.
    }

    broadcastLogoutSync();
    await setLogoutStatus();
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
    sessionStorage.clear();
    setRedirect(null);
    removeApiErrorInterceptor();
    removeApiAuthInterceptor();
    if (isSsoEnabled) {
      await userManager.clearStaleState();
      await userManager.removeUser();
    }
    stopAuthorizerFetchLoop();
  }, [isSsoEnabled, userManager]);

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
    if (typeof BroadcastChannel !== 'undefined') {
      logoutSyncRef.current = new BroadcastChannel(LOGOUT_SYNC_CHANNEL);
    }

    const handleSyncLogout = async () => {
      await setLogoutStatus();
    };

    const handleChannelMessage = async (event: MessageEvent) => {
      if (event.data?.type !== 'logout') {
        return;
      }

      if (event.data.sender === tabIdRef.current) {
        return;
      }

      await handleSyncLogout();
    };

    const handleStorageEvent = async (event: StorageEvent) => {
      if (event.key !== LOGOUT_SYNC_STORAGE_KEY || !event.newValue) {
        return;
      }

      try {
        const payload = JSON.parse(event.newValue);
        if (payload?.type === 'logout' && payload.sender !== tabIdRef.current) {
          localStorage.removeItem(LOGOUT_SYNC_STORAGE_KEY);
          await handleSyncLogout();
        }
      } catch {
        // Ignore malformed sync payloads.
      }
    };

    logoutSyncRef.current?.addEventListener('message', handleChannelMessage);
    window.addEventListener('storage', handleStorageEvent);

    return () => {
      logoutSyncRef.current?.removeEventListener(
        'message',
        handleChannelMessage
      );
      window.removeEventListener('storage', handleStorageEvent);
      logoutSyncRef.current?.close();
      logoutSyncRef.current = null;
    };
  }, [setLogoutStatus]);

  useEffect(() => {
    if (!isSsoEnabled) {
      return;
    }

    const handleUserLoaded = (user: { access_token?: string }) => {
      const token = user.access_token;
      if (!token) {
        return;
      }
      localStorage.setItem('everestToken', token);
      const decoded = jwtDecode(token);
      setLoggedInStatus(decoded.sub || '');
    };

    const handleTokenExpiring = () => {
      silentlyRenewToken();
    };

    userManager.events.addUserLoaded(handleUserLoaded);
    userManager.events.addAccessTokenExpiring(handleTokenExpiring);

    // signinSilentCallback is only relevant inside the silent-renew iframe.
    if (window.location !== window.parent.location) {
      userManager.signinSilentCallback();
    }

    return () => {
      userManager.events.removeUserLoaded(handleUserLoaded);
      userManager.events.removeAccessTokenExpiring(handleTokenExpiring);
    };
  }, [isSsoEnabled, silentlyRenewToken, userManager]);

  useEffect(() => {
    if (window.location !== window.parent.location) {
      // This is running in the iframe, so we are renewing the token silently
      return;
    }

    if (
      authStatus === 'loggedIn' ||
      authStatus === 'loggingIn' ||
      authStatus === 'loggedOut'
    ) {
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
          enqueueSnackbar('Your session has expired. Please sign in again.', {
            variant: 'info',
          });
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
