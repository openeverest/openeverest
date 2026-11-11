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

import { Alert } from '@mui/material';
import { useFormState } from 'react-hook-form';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { restartsPods } from 'utils/monitoring-classes';
import {
  getAttachedInstancesCount,
  getDestinationName,
} from 'utils/monitoring-destinations';
import { useDestinationInstances } from '../hooks/use-destination-instances';
import { DestinationFormFields } from '../monitoring-destination-modal.constants';
import { Messages } from '../monitoring-destination-modal.messages';
import { MonitoringDestinationFormValues } from '../monitoring-destination-modal.types';

interface RestartNoticeProps {
  destination: MonitoringDestination;
  monitoringClass: MonitoringClass | undefined;
}

// Providers roll the pods on new parameters or credentials; none reads the TLS settings.
export const RestartNotice = ({
  destination,
  monitoringClass,
}: RestartNoticeProps) => {
  const { dirtyFields } = useFormState<MonitoringDestinationFormValues>({
    name: [
      DestinationFormFields.spec,
      DestinationFormFields.fallbackParameters,
      DestinationFormFields.authMode,
      DestinationFormFields.credentialsSecretName,
      DestinationFormFields.credentials,
    ],
  });
  const readableInstances = useDestinationInstances(
    destination.metadata?.namespace ?? '',
    getDestinationName(destination)
  );
  const count = getAttachedInstancesCount(destination, readableInstances);
  const changesReachPods =
    !!dirtyFields.spec ||
    !!dirtyFields.fallbackParameters ||
    !!dirtyFields.authMode ||
    !!dirtyFields.credentialsSecretName ||
    !!dirtyFields.credentials;

  if (
    !changesReachPods ||
    !monitoringClass ||
    !restartsPods(monitoringClass) ||
    count === 0
  ) {
    return null;
  }

  return (
    <Alert severity="warning" data-testid="monitoring-destination-restart">
      {Messages.restartsInstances(count)}
    </Alert>
  );
};
