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

import { Box, BoxProps, Stack, Typography } from '@mui/material';

type Props = {
  title?: React.ReactNode;
  children: React.ReactNode;
  boxProps?: BoxProps;
};

const RoundedBox = ({ title, children, boxProps }: Props) => {
  const { sx, ...restProps } = boxProps || {};

  return (
    <Box
      className="percona-rounded-box"
      sx={[
        (theme) => ({
          p: 2,
          borderStyle: 'solid',
          borderWidth: '1px',
          borderColor: theme.palette.divider,
          borderRadius: 2,
        }),
        ...(sx ? (Array.isArray(sx) ? sx : [sx]) : []),
      ]}
      {...restProps}
    >
      <Stack>
        {title &&
          (typeof title === 'string' ? (
            <Typography variant="sectionHeading">{title}</Typography>
          ) : (
            title
          ))}
        <Box>{children}</Box>
      </Stack>
    </Box>
  );
};

export default RoundedBox;
