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
  Avatar,
  Box,
  Card,
  CardActionArea,
  CardContent,
  Tooltip,
  Typography,
} from '@mui/material';
import HelpOutlineIcon from '@mui/icons-material/HelpOutlineOutlined';
import { Link } from 'react-router-dom';
import { useExtensionCatalog } from 'hooks/api/extension-catalog';
import type { Provider } from 'shared-types/api.types';

type ProviderTilesProps = {
  providers: Provider[];
  showImport?: boolean;
};

const ProviderTiles = ({
  providers,
  showImport = false,
}: ProviderTilesProps) => {
  const { getProviderMeta } = useExtensionCatalog();

  return (
    <Box
      sx={{
        display: 'flex',
        flexWrap: 'wrap',
        justifyContent: 'center',
        gap: 2,
        width: '100%',
        maxWidth: 900,
      }}
    >
      {providers.map((provider) => {
        const name = provider?.metadata?.name ?? '';
        const meta = getProviderMeta(name);
        const label = meta?.displayName || name;
        return (
          <Card
            key={name}
            variant="outlined"
            sx={{
              flex: '1 1 200px',
              minWidth: 200,
              maxWidth: 220,
              display: 'flex',
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
              sx={{ height: '100%', display: 'flex' }}
            >
              <CardContent
                sx={{
                  flexGrow: 1,
                  display: 'flex',
                  flexDirection: 'row',
                  alignItems: 'center',
                  gap: 1.5,
                  py: 2,
                  px: 2,
                }}
              >
                {/* Enriched visuals only when catalog metadata is available
                    (plugin-hub installed and reachable). Otherwise fall back to
                    showing just the provider name. */}
                {meta && (
                  <Avatar
                    src={meta.icon}
                    variant="rounded"
                    sx={{
                      width: 36,
                      height: 36,
                      flexShrink: 0,
                      // Only tint the avatar when falling back to a letter;
                      // a real icon should render without a background.
                      bgcolor: meta.icon
                        ? 'transparent'
                        : (theme) => theme.palette.primary.main,
                      fontSize: '1rem',
                      textTransform: 'uppercase',
                    }}
                  >
                    {label.charAt(0)}
                  </Avatar>
                )}
                <Typography
                  variant="subtitle1"
                  fontWeight={400}
                  sx={{ flexGrow: 1, textAlign: 'left' }}
                >
                  {label}
                </Typography>
                {meta?.description && (
                  <Tooltip title={meta.description} placement="top" arrow>
                    <HelpOutlineIcon
                      fontSize="small"
                      sx={{ flexShrink: 0, color: 'text.secondary' }}
                      // Prevent the tooltip icon from triggering navigation.
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                      }}
                    />
                  </Tooltip>
                )}
              </CardContent>
            </CardActionArea>
          </Card>
        );
      })}
    </Box>
  );
};

export default ProviderTiles;
