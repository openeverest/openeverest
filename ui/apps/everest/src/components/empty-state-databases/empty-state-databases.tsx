import { Box, Link, Typography } from '@mui/material';
import CreateDbButton from 'components/create-db-button/create-db-button';
import EmptyState from 'components/empty-state';
import { Messages } from './messages';

const EmptyStateDatabases = ({
  showCreationButton,
  hasCreatePermission,
}: {
  showCreationButton: boolean;
  hasCreatePermission: boolean;
}) => {
  return (
    <>
      <EmptyState
        contentSlot={
          <>
            <Typography>{Messages.noDbClusters}</Typography>
            {hasCreatePermission ? (
              <Typography> {Messages.createToStart} </Typography>
            ) : (
              <>
                <Typography>{Messages.noPermissions}</Typography>
                <Typography>
                  Click{' '}
                  <Link
                    target="_blank"
                    rel="noopener"
                    href="https://openeverest.io/documentation/current/administer/rbac.html"
                  >
                    here
                  </Link>{' '}
                  to learn how to get permissions.
                </Typography>
              </>
            )}
          </>
        }
        showCreationButton={showCreationButton}
        buttonSlot={
          <Box display="flex" mb={1}>
            <CreateDbButton createFromImport />
            <CreateDbButton />
          </Box>
        }
      />
    </>
  );
};

export default EmptyStateDatabases;
