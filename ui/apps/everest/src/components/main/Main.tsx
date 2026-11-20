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

import { Box, Toolbar } from '@mui/material';
import useLocalStorage from 'hooks/utils/useLocalStorage';
import { GenericError } from 'pages/generic-error/GenericError';
import { useContext } from 'react';
import { Outlet } from 'react-router-dom';
import { ErrorBoundary } from 'utils/ErrorBoundary';
import { ErrorContextProvider } from 'utils/ErrorBoundaryProvider';
import { AppBar } from '../app-bar/AppBar';
import { Drawer } from '../drawer/Drawer';
import { WelcomeDialog } from '../welcome-dialog/welcome-dialog';
import UpgradeEverestReloadDialog from 'modals/upgrade-reload-everest-dialog';
import { UpgradeEverestContext } from 'contexts/upgrade-everest';

export const Main = () => {
  const [openWelcomeDialogLS, setOpenWelcomeDialogLS] = useLocalStorage(
    'welcomeModal',
    true
  );
  const { apiVersion, openReloadDialog, setOpenReloadDialog } = useContext(
    UpgradeEverestContext
  );

  const handleCloseWelcomeDialog = () => {
    setOpenWelcomeDialogLS(false);
  };

  return (
    <ErrorContextProvider>
      <ErrorBoundary fallback={<GenericError />}>
        <Box sx={{ display: 'flex' }}>
          <AppBar />
          <Drawer />
          <Box
            component="main"
            sx={{
              padding: 4,
              flexGrow: 1,
              minWidth: 0,
              overflow: 'hidden',
            }}
          >
            <Toolbar />
            <Outlet />
            {openWelcomeDialogLS && (
              <WelcomeDialog
                open={openWelcomeDialogLS}
                closeDialog={handleCloseWelcomeDialog}
              />
            )}
            <UpgradeEverestReloadDialog
              isOpen={openReloadDialog}
              closeModal={() => setOpenReloadDialog(false)}
              version={apiVersion || ''}
            />
          </Box>
        </Box>
      </ErrorBoundary>
    </ErrorContextProvider>
  );
};
