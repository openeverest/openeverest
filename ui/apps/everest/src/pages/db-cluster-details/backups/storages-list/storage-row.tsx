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

import { Box, Paper, Typography } from '@mui/material';
import { StorageRowProps } from './storage-row.types';

export const StorageRow = ({ storage }: StorageRowProps) => {
  const storageName = storage.storageRef.name;

  return (
    <Paper
      data-testid={`storage-row-${storageName}`}
      sx={{
        py: 1.5,
        px: 2,
        borderRadius: 1,
        boxShadow: 'none',
      }}
    >
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'row',
          alignItems: 'center',
        }}
      >
        <Typography variant="body1">{storageName}</Typography>
      </Box>
    </Paper>
  );
};
