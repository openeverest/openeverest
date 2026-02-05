import { Add } from '@mui/icons-material';
import { Button, Typography } from '@mui/material';
import { Table } from '@percona/ui-lib';
import EmptyState from 'components/empty-state';
import {
  useCreateLoadBalancerConfig,
  useDeleteLoadBalancerConfig,
  useLoadBalancerConfigs,
} from 'hooks/api/load-balancer';
import { useRBACPermissions } from 'hooks/rbac';
import { useNavigate } from 'react-router-dom';
import { useRef, useState } from 'react';
import LoadBalancerDialog from '../load-balancer-dialog';
import LoadBalancerRowActions from '../load-balancer-row-actions';
import DeleteLoadBalancerConfig from '../load-balancer-dialog/delete';
import { LoadBalancerConfig } from 'shared-types/loadbalancer.types';
import { useQueryClient } from '@tanstack/react-query';
import { messages } from '../load-balancer.messages';
import { EVEREST_READ_ONLY_FINALIZER } from 'consts';

const LoadBalancerConfigurationList = () => {
  const { data: loadBalancerConfigs, isLoading: loadBalancerConfigsLoading } =
    useLoadBalancerConfigs('load-balancer-configs', undefined, {
      refetchInterval: 10000,
    });
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const selectedConfig = useRef<LoadBalancerConfig | null>(null);
  const queryClient = useQueryClient();

  const navigate = useNavigate();
  const { canCreate } = useRBACPermissions('load-balancer-configs');
  const { mutate: createConfiguration } = useCreateLoadBalancerConfig(
    'create-load-balancer-config'
  );
  const { mutate: deleteConfiguration } = useDeleteLoadBalancerConfig(
    'delete-load-balancer-config'
  );

  const columns = [
    {
      accessorKey: 'metadata.name',
      header: 'Name',
    },
  ];

  const handleOnCreateConfiguration = (data: { name: string }) => {
    setDialogOpen(false);
    createConfiguration(data.name, {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: ['load-balancer-configs'],
        });
        navigate(
          `/settings/policies/details/load-balancer-configuration/${data.name}`,
          {
            state: {
              creation: true,
            },
          }
        );
      },
    });
  };

  const handleOnDeleteIconClick = (config: LoadBalancerConfig) => {
    selectedConfig.current = config;
    setDeleteDialogOpen(true);
  };

  const handleCloseDeleteDialog = () => {
    setDeleteDialogOpen(false);
    selectedConfig.current = null;
  };

  const handleOnDeleteConfig = () => {
    if (selectedConfig.current && selectedConfig.current.metadata) {
      deleteConfiguration(selectedConfig.current.metadata.name, {
        onSuccess: () => {
          queryClient.invalidateQueries({
            queryKey: ['load-balancer-configs'],
          });
        },
      });
      handleCloseDeleteDialog();
    }
  };

  return (
    <>
      <Table
        state={{ isLoading: loadBalancerConfigsLoading }}
        tableName="load-balancer-configuration"
        emptyState={
          <EmptyState
            onButtonClick={() => setDialogOpen(true)}
            buttonText={messages.configurationList.createButton}
            showCreationButton={canCreate}
            contentSlot={
              <Typography variant="body1">
                {messages.configurationList.emptyState.contentSlot}
              </Typography>
            }
          />
        }
        data={loadBalancerConfigs?.items || []}
        columns={columns}
        enableRowActions
        enableRowHoverAction
        rowHoverAction={(row) =>
          navigate(
            `/settings/policies/details/load-balancer-configuration/${row.original.metadata?.name}`
          )
        }
        renderTopToolbarCustomActions={() =>
          canCreate && loadBalancerConfigs?.items.length ? (
            <Button
              size="medium"
              startIcon={<Add />}
              data-testid="add-config"
              variant="contained"
              onClick={() => setDialogOpen(true)}
              sx={{ display: 'flex' }}
            >
              {messages.configurationList.createButton}
            </Button>
          ) : null
        }
        renderRowActions={({ row }) => (
          <LoadBalancerRowActions
            configName={row.original.metadata?.name ?? ''}
            handleOnDeleteIconClick={() =>
              handleOnDeleteIconClick(row.original)
            }
            readOnly={row.original.metadata.finalizers?.includes(
              EVEREST_READ_ONLY_FINALIZER
            )}
          />
        )}
      />
      {dialogOpen && (
        <LoadBalancerDialog
          open
          existingConfigs={loadBalancerConfigs?.items || []}
          onClose={() => setDialogOpen(false)}
          onSubmit={handleOnCreateConfiguration}
        />
      )}
      {deleteDialogOpen && (
        <DeleteLoadBalancerConfig
          isOpen
          configName={selectedConfig?.current?.metadata?.name ?? ''}
          handleConfirmDelete={handleOnDeleteConfig}
          handleCloseDeleteDialog={handleCloseDeleteDialog}
        />
      )}
    </>
  );
};

export default LoadBalancerConfigurationList;
