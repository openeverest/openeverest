import { NodeProps } from '@xyflow/react';
import { useNavigate, useParams } from 'react-router-dom';
import { CustomNode } from './types';
import { DBClusterComponent } from 'shared-types/components.types';
import {
  Box,
  Chip,
  IconButton,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import { COMPONENT_NODE_HEIGHT, COMPONENT_NODE_WIDTH } from './constants';
import ComponentStatus from '../component-status';
import {
  componentStatusToBaseStatus,
  COMPONENT_STATUS,
} from '../components.constants';
import DiagramComponentAge from './diagram-component-age';
import DiagramNode from './diagram-node';

const ComponentNode = ({
  data: {
    selected,
    componentData: { name, status, ready, type, restarts, started = '' },
  },
}: NodeProps<CustomNode<DBClusterComponent>>) => {
  const { dbClusterName = '', namespace = '' } = useParams();
  const navigate = useNavigate();
  const isPending = status === COMPONENT_STATUS.PENDING;

  const handleViewLogs = (e: React.MouseEvent) => {
    e.stopPropagation();
    const params = new URLSearchParams();
    params.set('component', name);
    navigate(
      `/databases/${namespace}/${dbClusterName}/logs?${params.toString()}`
    );
  };

  return (
    <DiagramNode
      height={COMPONENT_NODE_HEIGHT}
      width={COMPONENT_NODE_WIDTH}
      elevation={selected ? 4 : 0}
      dataTestId={`component-node-${name}${selected ? '-selected' : ''}`}
      showBottomHandle
      paperProps={{
        sx: {
          cursor: 'pointer',
        },
      }}
    >
      <Stack direction={'row'} alignItems={'center'}>
        <ComponentStatus
          status={status}
          statusMap={componentStatusToBaseStatus(ready)}
        />
        <Typography ml={'auto'} variant="body1">
          {ready} Ready
        </Typography>
      </Stack>
      <Stack mt={2}>
        <Tooltip title={name} placement="right" arrow>
          <Typography
            variant="body1"
            data-testid="component-name"
            sx={{
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {name}
          </Typography>
        </Tooltip>
        <DiagramComponentAge
          date={started}
          restarts={restarts}
          typographyProps={{
            variant: 'body2',
          }}
        />
      </Stack>
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          mt: 'auto',
        }}
      >
        <Chip label={type} data-testid="component-type" />
        {!isPending && (
          <Tooltip title="View Logs">
            <IconButton
              onClick={handleViewLogs}
              data-testid={`view-logs-${name}`}
              size="small"
              aria-label="View logs"
            >
              <VisibilityOutlinedIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        )}
      </Box>
    </DiagramNode>
  );
};

export default ComponentNode;
