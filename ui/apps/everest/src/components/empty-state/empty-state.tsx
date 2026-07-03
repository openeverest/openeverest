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
  Button,
  Divider,
  Typography,
  Link,
  Stack,
  ButtonProps,
} from '@mui/material';
import HelpIcon from '@mui/icons-material/HelpOutlineOutlined';
import { EmptyStateIcon } from '@percona/ui-lib';

const ContactSupportLink = ({ msg }: { msg: string }) => {
  return (
    <Link
      target="_blank"
      rel="noopener noreferrer"
      href="https://openeverest.io/support/"
    >
      <Button
        startIcon={
          <HelpIcon
            sx={{
              borderRadius: '10px',
            }}
          />
        }
      >
        {msg}
      </Button>
    </Link>
  );
};

type EmptyStateProps = {
  showCreationButton?: boolean;
  contentSlot?: React.ReactNode;
  buttonSlot?: React.ReactNode;
  buttonProps?: ButtonProps;
  buttonText?: string;
  onButtonClick?: () => void;
};

const EmptyState = ({
  showCreationButton = true,
  contentSlot,
  buttonSlot,
  buttonProps,
  buttonText,
  onButtonClick = () => {},
}: EmptyStateProps) => {
  return (
    <>
      <Stack
        alignItems="center"
        sx={{
          flexDirection: 'column',
          backgroundColor: (theme) =>
            theme.palette.surfaces?.elevation0 || 'transparent',
          p: 3,
          gap: 2,
        }}
      >
        <EmptyStateIcon w="60px" h="60px" />
        <Stack alignItems="center">
          {contentSlot ? contentSlot : <Typography>No data to show</Typography>}
        </Stack>
        {buttonSlot
          ? buttonSlot
          : showCreationButton && (
              <Button
                variant="contained"
                onClick={onButtonClick}
                {...buttonProps}
              >
                {buttonText || 'Create'}
              </Button>
            )}

        <Divider sx={{ width: '30%', marginTop: '10px' }} />
        <ContactSupportLink msg="Get support" />
      </Stack>
    </>
  );
};

export default EmptyState;
