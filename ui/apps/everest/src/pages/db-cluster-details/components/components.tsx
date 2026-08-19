// everest
// Copyright (C) 2023 Percona LLC
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

import { useDbClusterComponents } from 'hooks/api/db-cluster/useDbClusterComponents';
import { useParams } from 'react-router-dom';
import { useState } from 'react';
import { Box, FormControlLabel, Stack, Switch, TextField } from '@mui/material';
import ComponentsDiagramView from './diagram-view/components-diagram-view';
import { ReactFlowProvider } from '@xyflow/react';
import ComponentsTableView from './table-view';

const Components = () => {
  const [tableView, setTableView] = useState(false);
  const { dbClusterName = '', namespace = '' } = useParams();
  const { data: components = [], isLoading } = useDbClusterComponents(
    namespace,
    dbClusterName!
  );
  const [searchQuery, setSearchQuery] = useState('');

  const filteredComponents = components.filter((component) => {
    const keys = Object.keys(component) as (keyof typeof component)[];

    for (const key of keys) {
      if (
        typeof component[key] === 'string' &&
        component[key].toLowerCase().includes(searchQuery.toLowerCase())
      ) {
        return true;
      }
    }
    return false;
  });

  return (
    <Stack>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          mb: 1,
          mt: 1,
        }}
      >
        {!tableView && (
          <TextField
            label="Search"
            variant="outlined"
            size="small"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search"
            sx={{ width: '380px' }}
          />
        )}
        <FormControlLabel
          sx={{ ml: 'auto' }}
          control={
            <Switch
              data-testid="switch-input-table-view"
              value={tableView}
              onChange={(_, checked) => setTableView(checked)}
            />
          }
          label="Table view"
        />
      </Box>
      {tableView ? (
        <ComponentsTableView
          components={components}
          isLoading={isLoading}
          dbClusterName={dbClusterName}
        />
      ) : (
        <Box
          sx={{
            height: '500px',
            backgroundColor: 'surfaces.elevation0',
            borderRadius: 2,
          }}
        >
          <ReactFlowProvider>
            <ComponentsDiagramView components={filteredComponents} />
          </ReactFlowProvider>
        </Box>
      )}
    </Stack>
  );
};
export default Components;
