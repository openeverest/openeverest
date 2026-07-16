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

import { Box, IconButton, Tooltip } from '@mui/material';
import InfoIcon from '@mui/icons-material/InfoOutlined';

const WithInfoIcon = ({
  children,
  onClick,
  disabled = false,
  tooltip,
}: {
  children: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  tooltip: string;
}) => {
  return (
    <Box
      sx={{
        display: 'flex',
        ml: 'auto',
        alignItems: 'center',
      }}
    >
      {children}
      <Tooltip title={tooltip} placement="right" arrow>
        <span>
          <IconButton
            onClick={onClick}
            disabled={disabled}
            sx={[
              disabled
                ? {
                    opacity: 0.5,
                  }
                : {
                    opacity: 1,
                  },
              disabled
                ? {
                    cursor: 'not-allowed',
                  }
                : {
                    cursor: 'pointer',
                  },
            ]}
          >
            <InfoIcon
              sx={[
                {
                  width: '20px',
                },
                disabled
                  ? {
                      color: 'GrayText',
                    }
                  : {
                      color: 'default',
                    },
              ]}
            />
          </IconButton>
        </span>
      </Tooltip>
    </Box>
  );
};

export default WithInfoIcon;
