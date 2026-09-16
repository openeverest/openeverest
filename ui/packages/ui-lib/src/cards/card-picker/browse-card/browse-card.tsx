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

import { Card, CardActionArea, Typography } from '@mui/material';

interface BrowseCardProps {
  label: string;
  onClick: () => void;
}

// Dashed tile that opens the full searchable picker for the overflow items.
export const BrowseCard = ({ label, onClick }: BrowseCardProps) => (
  <Card variant="dashed" sx={{ display: 'flex' }}>
    <CardActionArea
      onClick={onClick}
      sx={{
        minHeight: 96,
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <Typography variant="sectionHeading" color="primary">
        {label}
      </Typography>
    </CardActionArea>
  </Card>
);
