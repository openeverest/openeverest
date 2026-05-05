import { Add } from '@mui/icons-material';
import { Button } from '@mui/material';
import { Table } from '@percona/ui-lib';
import { useMemo, useRef, useState } from 'react';
import {
  useCreateSplitHorizonConfig,
  useDeleteSplitHorizonDNSConfig,
  useNamespaces,
  useUpdateSplitHorizonConfig,
} from 'hooks/api';
import { useQueries, useQueryClient } from '@tanstack/react-query';
import { getAllSplitHorizonDNSConfigs } from 'api/splitHorizon';
import ConfigurationModal from './configuration-modal';
import { fileToBase64 } from 'utils/db';
import SplitHorizonRowActions from './row-actions';
import { TableRow } from './types';
import DeleteSplitHorizonConfigDialog from './delete-dialog';
import { useNamespacePermissionsForResource } from 'hooks/rbac';

const SplitHorizon = () => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [removeDialogOpen, setRemoveDialogOpen] = useState(false);
  const selectedConfig = useRef<TableRow | undefined>();
  const { canCreate } = useNamespacePermissionsForResource(
    'enginefeatures/split-horizon-dns-configs'
  );
  const queryClient = useQueryClient();
  const { mutate: createSplitHorizonConfig, isPending: isCreating } =
    useCreateSplitHorizonConfig({
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: ['split-horizon-configs'],
        });
        setIsModalOpen(false);
      },
    });
  const { mutate: updateSplitHorizonConfig, isPending: isUpdating } =
    useUpdateSplitHorizonConfig({
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: ['split-horizon-configs'],
        });
        selectedConfig.current = undefined;
        setIsModalOpen(false);
      },
    });
  const { mutate: deleteSplitHorizonConfig, isPending: isDeleting } =
    useDeleteSplitHorizonDNSConfig({
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: ['split-horizon-configs'],
        });
        setRemoveDialogOpen(false);
        selectedConfig.current = undefined;
      },
    });
  const { data: namespaces = [] } = useNamespaces({
    refetchInterval: 10 * 1000,
  });
  const splitHorizonConfigs = useQueries({
    queries: namespaces.map((ns) => ({
      queryKey: ['split-horizon-configs', ns],
      queryFn: () => getAllSplitHorizonDNSConfigs(ns),
      refetchInterval: 10 * 1000,
    })),
  });

  const columns = useMemo(
    () => [
      {
        header: 'Name',
        accessorKey: 'name',
      },
      {
        header: 'Domain name',
        accessorKey: 'domain',
      },
      {
        header: 'Namespace',
        accessorKey: 'namespace',
      },
    ],
    []
  );

  const data = useMemo<TableRow[]>(() => {
    return splitHorizonConfigs.flatMap(({ data }) =>
      (data || []).map((config) => ({
        name: config.metadata.name,
        domain: config.spec.baseDomainNameSuffix,
        namespace: config.namespace,
        secretName: config.spec.tls.secretName,
        inUse: config.status.inUse,
      }))
    );
  }, [splitHorizonConfigs]);

  const handleOnEditIconClick = (config: TableRow) => {
    selectedConfig.current = config;
    setIsModalOpen(true);
  };

  const handleOnDeleteIconClick = (config: TableRow) => {
    selectedConfig.current = config;
    setRemoveDialogOpen(true);
  };

  const handleConfirmDelete = () => {
    deleteSplitHorizonConfig({
      name: selectedConfig.current!.name,
      namespace: selectedConfig.current!.namespace,
    });
  };

  const handleOnCloseModal = () => {
    setIsModalOpen(false);
    selectedConfig.current = undefined;
  };

  const handleModalConfirm = (
    name: string,
    namespace: string,
    baseDomain: string,
    caCrt: string | undefined,
    caKey: string | undefined,
    secretName: string
  ) => {
    if (selectedConfig.current) {
      updateSplitHorizonConfig({
        name,
        namespace,
        baseDomain,
        caCrt,
        caKey,
        secretName,
      });
    } else {
      createSplitHorizonConfig({
        name,
        namespace,
        baseDomain,
        caCrt: caCrt ?? '',
        caKey: caKey ?? '',
        secretName,
      });
    }
  };

  return (
    <>
      <Table
        tableName="split-horizon"
        columns={columns}
        data={data}
        enableRowActions
        renderTopToolbarCustomActions={() =>
          canCreate.length > 0 && (
            <Button
              variant="contained"
              size="medium"
              onClick={() => setIsModalOpen(true)}
              data-testid="add-config-button"
              sx={{ display: 'flex' }}
              startIcon={<Add />}
            >
              Create configuration
            </Button>
          )
        }
        renderRowActions={({ row }) => (
          <SplitHorizonRowActions
            namespace={row.original.namespace}
            configName={row.original.name}
            isConfigInUse={row.original.inUse}
            handleOnDeleteIconClick={() =>
              handleOnDeleteIconClick(row.original)
            }
            handleOnEditIconClick={() => handleOnEditIconClick(row.original)}
          />
        )}
      />
      {isModalOpen && (
        <ConfigurationModal
          isOpen
          isSubmitting={isCreating || isUpdating}
          selectedConfig={selectedConfig.current}
          namespacesAvailable={canCreate}
          onClose={handleOnCloseModal}
          onSubmit={async (data) => {
            handleModalConfirm(
              data.name,
              data.namespace,
              data.domain,
              data.caCert ? await fileToBase64(data.caCert) : undefined,
              data.caKey ? await fileToBase64(data.caKey) : undefined,
              data.secretName
            );
          }}
        />
      )}
      {removeDialogOpen && (
        <DeleteSplitHorizonConfigDialog
          configName={selectedConfig.current!.name}
          configInUse={selectedConfig.current!.inUse}
          handleCloseDeleteDialog={() => setRemoveDialogOpen(false)}
          handleConfirmDelete={handleConfirmDelete}
          submitting={isDeleting}
        />
      )}
    </>
  );
};

export default SplitHorizon;
