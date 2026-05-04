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

import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import ActionableAlert from 'components/actionable-alert';
import { CreateEditEndpointModal } from 'pages/settings/monitoring-endpoints/createEditModal/create-edit-modal';
import { EndpointFormType } from 'pages/settings/monitoring-endpoints/createEditModal/create-edit-modal.types';
import {
  MONITORING_CONFIGS_QUERY_KEY,
  useCreateMonitoringConfig,
} from 'hooks/api/monitoring/useMonitoringConfigsList';
import { useRBACPermissions } from 'hooks/rbac';
import type { EmptyStateFallbackProps } from '../types';
import { Messages } from './monitoring-empty-fallback.messages';

export const MonitoringEmptyFallback = ({
  namespace,
}: EmptyStateFallbackProps) => {
  const [openModal, setOpenModal] = useState(false);
  const queryClient = useQueryClient();
  const { mutate: createMonitoringConfig, isPending } =
    useCreateMonitoringConfig();
  const { canCreate } = useRBACPermissions(
    'monitoring-configs',
    `${namespace}/*`
  );

  const handleSubmit = (_isEditMode: boolean, data: EndpointFormType) => {
    const { name, url, verifyTLS, user, password } = data;
    const pmm: Record<string, string> = {};
    if (user) pmm.user = user;
    if (password) pmm.password = password;

    createMonitoringConfig(
      {
        namespace,
        payload: { name, url, type: 'pmm', verifyTLS, pmm },
      },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({
            queryKey: [MONITORING_CONFIGS_QUERY_KEY],
          });
          setOpenModal(false);
        },
      }
    );
  };

  return (
    <>
      <ActionableAlert
        message={Messages.alertText(namespace)}
        buttonMessage={Messages.addMonitoringEndpoint}
        data-testid="monitoring-empty-fallback"
        onClick={() => setOpenModal(true)}
        {...(!canCreate && { action: undefined })}
      />
      {openModal && (
        <CreateEditEndpointModal
          open={openModal}
          handleClose={() => setOpenModal(false)}
          handleSubmit={handleSubmit}
          isLoading={isPending}
          selectedNamespace={namespace}
        />
      )}
    </>
  );
};
