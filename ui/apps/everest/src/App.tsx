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

import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFnsV3';
import createCache from '@emotion/cache';
import { CacheProvider } from '@emotion/react';
import { ThemeContextProvider, everestThemeOptions } from '@percona/design';
import { NotistackMuiSnackbar } from '@percona/ui-lib';
import { SnackbarProvider } from 'notistack';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { RouterProvider } from 'react-router-dom';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { AuthProvider } from 'contexts/auth';
import { DrawerContextProvider } from 'contexts/drawer/drawer.context';
import router from 'router';
import { useEffect, useState } from 'react';
import { EverestConfig } from 'shared-types/configs.types';
import { getEverestConfigs } from 'api/everestConfigs';
import LoadingPageSkeleton from 'components/loading-page-skeleton/LoadingPageSkeleton';
import UpgradeEverestProvider from 'contexts/upgrade-everest/upgrade-everest.provider';
import { PluginProvider } from 'contexts/plugins';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: (failureCount, error) =>
        (error as AxiosError).status === 429 ? failureCount < 4 : false,
    },
  },
});

const App = () => {
  const [configs, setConfigs] = useState<EverestConfig | undefined | null>();

  useEffect(() => {
    const loadConfigs = async () => {
      try {
        const { oidcConfig = { clientId: '', issuerURL: '', scopes: [] } } =
          await getEverestConfigs();
        setConfigs({
          oidc: {
            authority: oidcConfig.issuerURL,
            clientId: oidcConfig.clientId,
            scope: oidcConfig.scopes.join(' '),
            redirectUri: `${window.location.protocol}//${window.location.host}/`,
          },
        });
      } catch (error) {
        setConfigs(null);
      }
    };

    loadConfigs();
  }, []);

  if (configs === undefined) {
    return <LoadingPageSkeleton />;
  }

  const nonce =
    document.querySelector("meta[name='csp-nonce']")?.getAttribute('content') ||
    '';

  const cache = createCache({
    key: 'percona-css',
    nonce,
    prepend: true,
  });

  return (
    <CacheProvider value={cache}>
      <ThemeContextProvider
        themeOptions={everestThemeOptions}
        saveColorModeOnLocalStorage
      >
        <LocalizationProvider dateAdapter={AdapterDateFns}>
          <SnackbarProvider
            maxSnack={3}
            preventDuplicate
            // NOTE: using custom components disables notistack's custom actions, as per docs: https://notistack.com/features/basic#actions
            // If we need actions, we can add them to our custom component via useSnackbar(): https://notistack.com/features/customization#custom-component
            Components={{
              success: NotistackMuiSnackbar,
              error: NotistackMuiSnackbar,
              info: NotistackMuiSnackbar,
              warning: NotistackMuiSnackbar,
            }}
          >
            <QueryClientProvider client={queryClient}>
              <AuthProvider
                oidcConfig={{
                  ...configs?.oidc,
                  redirectUri: `${window.location.protocol}//${window.location.host}/login-callback`,
                  responseType: 'code',
                  autoSignIn: false,
                  automaticSilentRenew: false,
                  loadUserInfo: false,
                }}
              >
                <DrawerContextProvider>
                  <PluginProvider>
                    <UpgradeEverestProvider>
                      <RouterProvider router={router} />
                    </UpgradeEverestProvider>
                  </PluginProvider>
                </DrawerContextProvider>
                <ReactQueryDevtools initialIsOpen={false} />
              </AuthProvider>
            </QueryClientProvider>
          </SnackbarProvider>
        </LocalizationProvider>
      </ThemeContextProvider>
    </CacheProvider>
  );
};

export default App;
