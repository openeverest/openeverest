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
            command="kubectl create namespace <NAMESPACE>"
          />
          <CommandInstructions
            message="If you are using Helm, run the following command:"
            command="helm install everest openeverest/everest-db-namespace --create-namespace --namespace <NAMESPACE>"
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
