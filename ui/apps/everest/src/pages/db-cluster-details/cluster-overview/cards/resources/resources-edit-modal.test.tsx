import { render, waitFor, fireEvent, screen } from '@testing-library/react';
import { DbType } from '@percona/types';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import ResourcesEditModal from './resources-edit-modal';
import { ResourceSize } from 'components/cluster-form';
const queryClient = new QueryClient();

describe('ResourcesEditModal', () => {
  it('should disable descaling to one node', async () => {
    const { getByTestId, getByText } = render(
      <QueryClientProvider client={queryClient}>
        <ResourcesEditModal
          allowDiskDescaling={false}
          storageClass="standard"
          handleCloseModal={vi.fn()}
          dbType={DbType.Mysql}
          shardingEnabled={false}
          onSubmit={vi.fn()}
          defaultValues={{
            numberOfNodes: '3',
            numberOfProxies: '3',
            cpu: 1,
            memory: 1,
            disk: 1,
            proxyCpu: 1,
            proxyMemory: 1,
            diskUnit: 'Gi',
            memoryUnit: 'Gi',
            resourceSizePerNode: ResourceSize.custom,
            resourceSizePerProxy: ResourceSize.custom,
          }}
        />
      </QueryClientProvider>
    );

    await waitFor(() =>
      expect(getByTestId('form-dialog-save')).not.toBeDisabled()
    );

    fireEvent.click(getByTestId('toggle-button-nodes-1'));

    await waitFor(() => expect(getByTestId('form-dialog-save')).toBeDisabled());
    expect(getByText('Cannot scale down to one node.')).toBeInTheDocument();

    fireEvent.click(getByTestId('toggle-button-nodes-5'));

    await waitFor(() =>
      expect(getByTestId('form-dialog-save')).not.toBeDisabled()
    );
  });

  it('should not allow to decrease disk unit and show error message', async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <ResourcesEditModal
          allowDiskDescaling={false}
          storageClass="standard"
          handleCloseModal={vi.fn()}
          dbType={DbType.Mysql}
          shardingEnabled={false}
          onSubmit={vi.fn()}
          defaultValues={{
            numberOfNodes: '3',
            numberOfProxies: '3',
            cpu: 1,
            memory: 1,
            disk: 2,
            proxyCpu: 1,
            proxyMemory: 1,
            diskUnit: 'Gi',
            memoryUnit: 'Gi',
            resourceSizePerNode: ResourceSize.custom,
            resourceSizePerProxy: ResourceSize.custom,
          }}
        />
      </QueryClientProvider>
    );

    const input = screen.getByTestId('text-input-disk');
    const button = screen.getByTestId('form-dialog-save');

    await waitFor(() => expect(input).toHaveValue('2'));

    fireEvent.change(input, { target: { value: '1' } });
    await waitFor(() => expect(button).toBeDisabled());
    await waitFor(() =>
      expect(screen.getByText('Descaling is not allowed')).toBeInTheDocument()
    );
  });

  it('should not block from increasing disk unit and show warning', async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <ResourcesEditModal
          allowDiskDescaling={false}
          storageClass="standard"
          handleCloseModal={vi.fn()}
          dbType={DbType.Mysql}
          shardingEnabled={false}
          onSubmit={vi.fn()}
          defaultValues={{
            numberOfNodes: '3',
            numberOfProxies: '3',
            cpu: 1,
            memory: 1,
            disk: 2,
            proxyCpu: 1,
            proxyMemory: 1,
            diskUnit: 'Gi',
            memoryUnit: 'Gi',
            resourceSizePerNode: ResourceSize.custom,
            resourceSizePerProxy: ResourceSize.custom,
          }}
        />
      </QueryClientProvider>
    );

    const input = screen.getByTestId('text-input-disk');
    const button = screen.getByTestId('form-dialog-save');

    await waitFor(() => expect(input).toHaveValue('2'));

    fireEvent.change(input, { target: { value: '3' } });
    await waitFor(() =>
      expect(
        screen.getByText(
          'Disk upscaling is irreversible and may temporarily block further resize actions until complete.'
        )
      ).toBeInTheDocument()
    );

    fireEvent.click(button);
  });

  it('should not allow to submit empty disk value', async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <ResourcesEditModal
          allowDiskDescaling={false}
          storageClass="standard"
          handleCloseModal={vi.fn()}
          dbType={DbType.Mysql}
          shardingEnabled={false}
          onSubmit={vi.fn()}
          defaultValues={{
            numberOfNodes: '3',
            numberOfProxies: '3',
            cpu: 1,
            memory: 1,
            disk: 2,
            proxyCpu: 1,
            proxyMemory: 1,
            diskUnit: 'Gi',
            memoryUnit: 'Gi',
            resourceSizePerNode: ResourceSize.custom,
            resourceSizePerProxy: ResourceSize.custom,
          }}
        />
      </QueryClientProvider>
    );

    const input = screen.getByTestId('text-input-disk');
    const button = screen.getByTestId('form-dialog-save');

    await waitFor(() => expect(input).toHaveValue('2'));

    fireEvent.change(input, { target: { value: '' } });
    await waitFor(() => expect(button).toBeDisabled());
    await waitFor(() =>
      expect(
        screen.getByText('String must contain at least 1 character(s)')
      ).toBeInTheDocument()
    );
  });

  it('should be disabled disk unit and show tooltip', async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <ResourcesEditModal
          allowDiskDescaling={false}
          storageClass="not_standard"
          handleCloseModal={vi.fn()}
          dbType={DbType.Mysql}
          shardingEnabled={false}
          onSubmit={vi.fn()}
          defaultValues={{
            numberOfNodes: '3',
            numberOfProxies: '3',
            cpu: 1,
            memory: 1,
            disk: 2,
            proxyCpu: 1,
            proxyMemory: 1,
            diskUnit: 'Gi',
            memoryUnit: 'Gi',
            resourceSizePerNode: ResourceSize.custom,
            resourceSizePerProxy: ResourceSize.custom,
          }}
        />
      </QueryClientProvider>
    );

    const input = screen.getByTestId('text-input-disk');

    await waitFor(() => expect(input).toBeDisabled());

    expect(screen.getByTestId('disk-tooltip')).toBeInTheDocument();
  });
});
