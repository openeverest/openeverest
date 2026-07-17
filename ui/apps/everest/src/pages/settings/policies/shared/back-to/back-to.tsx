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

import ArrowBackIcon from '@mui/icons-material/ArrowBack';

import { Box, Typography } from '@mui/material';
import { Link } from 'react-router-dom';
import { messages } from '../../policies.messages';

interface BackToProps {
  to: string;
  prevPage: string;
}

const BackTo = ({ to, prevPage }: BackToProps) => {
  return (
    <Link to={to} style={{ textDecoration: 'none' }}>
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'start',
          alignItems: 'center',
          gap: 1,
          mb: 2,
        }}
      >
        <ArrowBackIcon sx={{ color: 'primary.main' }} />
        <Typography
          variant="body2"
          sx={{
            color: 'primary.main',
          }}
        >
          {messages.backTo.back} {prevPage}
        </Typography>
      </Box>
    </Link>
  );
};
export default BackTo;
