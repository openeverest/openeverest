import { NodeProps } from '@xyflow/react';
import { CustomNode } from './types';
import { Container } from 'shared-types/components.types';
import { Stack, Typography } from '@mui/material';
import { CONTAINER_NODE_HEIGHT, CONTAINER_NODE_WIDTH } from './constants';
import { containerStatusToBaseStatus } from '../components.constants';
import ComponentStatus from '../component-status';
import DiagramComponentAge from './diagram-component-age';
import DiagramNode from './diagram-node';

const ContainerNode = ({
  data: {
    componentData: { status, ready, name, started, restarts },
  },
}: NodeProps<CustomNode<Container>>) => (
  <DiagramNode
    height={CONTAINER_NODE_HEIGHT}
    width={CONTAINER_NODE_WIDTH}
    dataTestId={`container-node-${name}`}
    showTopHandle
  >
    <Stack direction={'row'} sx={{
      alignItems: 'center'
    }}>
      <ComponentStatus
        status={status}
        statusMap={containerStatusToBaseStatus(ready)}
      />
      <Typography variant="body1" sx={{
        ml: 'auto'
      }}>
        {ready ? 'Ready' : 'Not Ready'}
      </Typography>
    </Stack>
    <Typography
      variant="body1"
      data-testid="container-name"
      sx={{
        whiteSpace: 'nowrap',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        mt: 2,
      }}
    >
      {name}
    </Typography>
    <DiagramComponentAge date={started} restarts={restarts} />
  </DiagramNode>
);

export default ContainerNode;
