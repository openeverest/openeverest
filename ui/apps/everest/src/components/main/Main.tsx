import ReplayIcon from '@mui/icons-material/Replay';
import { Box, Button, Stack, Toolbar, Typography } from '@mui/material';
import { useKubernetesClusterInfo } from 'hooks/api/kubernetesClusters/useKubernetesClusterInfo';
import useLocalStorage from 'hooks/utils/useLocalStorage';
import { GenericError } from 'pages/generic-error/GenericError';
import { useContext } from 'react';
import { Outlet } from 'react-router-dom';
import { ErrorBoundary } from 'utils/ErrorBoundary';
import { ErrorContextProvider } from 'utils/ErrorBoundaryProvider';
import { AppBar } from '../app-bar/AppBar';
import { Drawer } from '../drawer/Drawer';
import { WelcomeDialog } from '../welcome-dialog/welcome-dialog';
import { Messages } from './Main.messages';
import LoadingPageSkeleton from 'components/loading-page-skeleton/LoadingPageSkeleton';
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
  const { isFetching, isError, refetch } = useKubernetesClusterInfo([
    'initial-k8-info',
  ]);

  const handleCloseWelcomeDialog = () => {
    setOpenWelcomeDialogLS(false);
  };

  const handleClick = () => {
    refetch();
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
            {isFetching ? (
              <LoadingPageSkeleton />
            ) : isError ? (
              <Stack alignItems="center">
                <Typography variant="subtitle1">
                  {Messages.somethingWrong}
                </Typography>
                <Button
                  onClick={handleClick}
                  variant="outlined"
                  endIcon={<ReplayIcon />}
                  sx={{ mt: 1 }}
                >
                  {Messages.retry}
                </Button>
              </Stack>
            ) : (
              <Outlet />
            )}
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
