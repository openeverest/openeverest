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

import {
  Box,
  Card,
  CardActionArea,
  CardContent,
  Typography,
} from '@mui/material';
import { Link } from 'react-router-dom';
import type { Provider } from 'shared-types/api.types';

type ProviderTilesProps = {
  providers: Provider[];
  showImport?: boolean;
};

const ProviderTiles = ({
  providers,
  showImport = false,
}: ProviderTilesProps) => {
  return (
    <Box
      sx={{
        display: 'grid',
        gap: 2,
        width: '100%',
        maxWidth: 720,
        gridTemplateColumns: {
          xs: '1fr',
          sm: 'repeat(2, 1fr)',
          md: 'repeat(3, 1fr)',
          lg: 'repeat(4, 1fr)',
        },
      }}
    >
      {providers.map((provider) => {
        const name = provider?.metadata?.name ?? '';
        return (
          <Card
            key={name}
            variant="outlined"
            sx={{
              borderRadius: 2,
              transition: 'border-color 0.15s, box-shadow 0.15s',
              '&:hover': {
                borderColor: (theme) => theme.palette.primary.main,
                boxShadow: 2,
              },
            }}
          >
            <CardActionArea
              data-testid={`provider-tile-${name}`}
              component={Link}
              to="/databases/new"
              state={{
                selectedDbProvider: provider,
                showImport,
              }}
              sx={{ height: '100%' }}
            >
              <CardContent
                sx={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  textAlign: 'center',
                  py: 3,
                  gap: 1,
                }}
              >
                <Typography variant="subtitle1" fontWeight={600}>
                  {name}
                </Typography>
              </CardContent>
            </CardActionArea>
          </Card>
        );
      })}
    </Box>
  );
};

export default ProviderTiles;
