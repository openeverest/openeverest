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

import { Box, Button, Typography } from '@mui/material';
import { NoMatchIcon } from '@percona/ui-lib';
import { useActiveBreakpoint } from 'hooks/utils/useActiveBreakpoint';
import { Link } from 'react-router-dom';
import { Messages } from './NoMatch.messages';
import { NoMatchProps } from './NoMatch.type';

export const NoMatch = ({
  header = Messages.header,
  subHeader = Messages.subHeader,
  redirectButtonText = Messages.redirectButton,
  CustomIcon,
  onButtonClick,
  customButton,
}: NoMatchProps) => {
  const { isMobile, isTablet, isDesktop } = useActiveBreakpoint();

  return (
    <Box
      sx={[
        {
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
        },
        isDesktop
          ? {
              height: '435px',
            }
          : {
              height: 'auto',
            },
        isDesktop
          ? {
              width: '980px',
            }
          : {
              width: 'auto',
            },
        isTablet
          ? {
              mt: '58px',
            }
          : {
              mt: isMobile ? '13px' : '150px',
            },
        isTablet
          ? {
              mx: '58px',
            }
          : {
              mx: isMobile ? '13px' : 'auto',
            },
        isDesktop
          ? {
              flexDirection: 'row',
            }
          : {
              flexDirection: 'column',
            },
      ]}
    >
      <Box>
        {CustomIcon ? (
          <CustomIcon
            w={isMobile ? '300px' : '435px'}
            h={isMobile ? '300px' : '435px'}
          />
        ) : (
          <NoMatchIcon
            w={isMobile ? '300px' : '435px'}
            h={isMobile ? '300px' : '435px'}
          />
        )}
      </Box>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
        <Typography
          sx={{
            fontWeight: 600,
            fontSize: '40px',
            lineHeight: '40px',
            letterSpacing: '-0.025em',
            fontFamily:
              '"Poppins", "Roboto", "Helvetica", "Arial", "sans-serif"',
          }}
        >
          {header}
        </Typography>
        <Typography
          sx={{
            fontWeight: 400,
            fontSize: '16px',
            lineHeight: '19.44px',
            letterSpacing: '-0.025em',
            fontFamily:
              '"Poppins", "Roboto", "Helvetica", "Arial", "sans-serif"',
          }}
        >
          {subHeader}
        </Typography>
        {customButton ? (
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'row',
              alignItems: 'center',
              gap: 1,
              mt: 2,
            }}
          >
            <Button
              component={Link}
              to="/"
              variant="contained"
              data-testid="no-match-button"
              onClick={onButtonClick}
            >
              {redirectButtonText}
            </Button>
            {customButton}
          </Box>
        ) : (
          <Button
            component={Link}
            to="/"
            sx={{ alignSelf: 'start', mt: 2 }}
            variant="contained"
            data-testid="no-match-button"
            onClick={onButtonClick}
          >
            {redirectButtonText}
          </Button>
        )}
      </Box>
    </Box>
  );
};
