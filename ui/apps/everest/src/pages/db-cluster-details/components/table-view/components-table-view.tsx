import { useMemo } from 'react';
import { MRT_ColumnDef } from 'material-react-table';
import { Table } from '@percona/ui-lib';
import { DBClusterComponent } from 'shared-types/components.types';
import {
  COMPONENT_STATUS,
  COMPONENT_STATUS_WEIGHT,
  componentStatusToBaseStatus,
} from '../components.constants';
import ExpandedRow from '../expanded-row';
import ComponentStatus from '../component-status';
import ComponentAge from '../component-age';
import { useParams, useNavigate } from 'react-router-dom';
import { IconButton, Tooltip } from '@mui/material';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';

const ComponentsTableView = ({
  components,
  isLoading,
  dbClusterName,
}: {
  components: DBClusterComponent[];
  isLoading: boolean;
  dbClusterName: string;
}) => {
  const { namespace = '' } = useParams();
  const navigate = useNavigate();

  const handleViewLogs = (componentName: string, containerName?: string) => {
    const params = new URLSearchParams();
    params.set('component', componentName);
    if (containerName) {
      params.set('container', containerName);
    }
    navigate(
      `/databases/${namespace}/${dbClusterName}/logs?${params.toString()}`
    );
  };

  const columns = useMemo<MRT_ColumnDef<DBClusterComponent>[]>(() => {
    return [
      {
        header: 'Status',
        accessorKey: 'status',
        Cell: ({ cell, row }) => (
          <ComponentStatus
            status={cell.getValue<COMPONENT_STATUS>()}
            statusMap={componentStatusToBaseStatus(row.original.ready)}
          />
        ),
        sortingFn: (rowA, rowB) => {
          return (
            COMPONENT_STATUS_WEIGHT[rowA?.original?.status] -
            COMPONENT_STATUS_WEIGHT[rowB?.original?.status]
          );
        },
      },
      {
        header: 'Ready',
        accessorKey: 'ready',
        Cell: ({ cell }) => (
          <span data-testid="component-ready-status">
            {cell.getValue<string>()}
          </span>
        ),
      },
      {
        header: 'Name',
        accessorKey: 'name',
      },
      {
        header: 'Type',
        accessorKey: 'type',
      },
      {
        header: 'Age',
        accessorKey: 'started',
        Cell: ({ cell }) => <ComponentAge date={cell.getValue<string>()} />,
      },
      {
        header: 'Restarts',
        accessorKey: 'restarts',
      },
    ];
  }, []);

  return (
    <Table
      data-testid="components-table-view"
      getRowId={(row) => row.name}
      initialState={{
        sorting: [
          {
            id: 'status',
            desc: true,
          },
        ],
      }}
      state={{ isLoading }}
      tableName={`${dbClusterName}-components`}
      columns={columns}
      data={components}
      noDataMessage={'No components'}
      renderDetailPanel={({ row }) => <ExpandedRow row={row} />}
      muiTableDetailPanelProps={{
        sx: {
          padding: 0,
          width: '100%',
          '.MuiCollapse-root': {
            width: '100%',
          },
        },
      }}
      enableRowActions
      renderRowActions={({ row }) => {
        const component = row.original;
        const isComponentPending =
          component.status === COMPONENT_STATUS.PENDING;

        return (
          !isComponentPending && (
            <Tooltip title="View Logs">
              <IconButton
                onClick={() => handleViewLogs(component.name)}
                data-testid={`view-logs-${component.name}`}
                size="small"
                aria-label="View logs"
              >
                <VisibilityOutlinedIcon />
              </IconButton>
            </Tooltip>
          )
        );
      }}
    />
  );
};

export default ComponentsTableView;
