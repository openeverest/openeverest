import { Button, Stack, Typography } from '@mui/material';
import { CodeCopyBlock } from '@percona/ui-lib';
import { ArrowOutward } from '@mui/icons-material';
import EmptyState from 'components/empty-state';
import { Messages } from './messages';

const CommandInstructions = ({
  message,
  command,
}: {
  message: string;
  command: string;
}) => (
  <Stack mt={3} maxWidth="350px">
    <Typography variant="body2">{message}</Typography>
    <CodeCopyBlock message={command} />
  </Stack>
);

const EmptyStateNamespaces = () => {
  return (
    <EmptyState
      contentSlot={
        <>
          <Typography>{Messages.noNamespaces}</Typography>
          <Typography> {Messages.createToStart}</Typography>
          <CommandInstructions
            message="If you are using CLI, run the following command:"
            command="everestctl namespaces add <NAMESPACE>"
          />
          <CommandInstructions
            message="If you are using Helm, run the following command:"
            command="helm install everest percona/everest-db-namespace --create-namespace --namespace <NAMESPACE>"
          />
        </>
      }
      buttonSlot={
        <Button
          data-testid="learn-more-button"
          size="small"
          variant="contained"
          sx={{ display: 'flex' }}
          onClick={() => {
            window.open(
              'https://openeverest.io/documentation/current/administer/manage_namespaces.html',
              '_blank',
              'noopener'
            );
          }}
          endIcon={<ArrowOutward />}
        >
          {Messages.learnMore}
        </Button>
      }
    />
  );
};

export default EmptyStateNamespaces;
