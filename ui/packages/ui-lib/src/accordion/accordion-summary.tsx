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

import { AccordionSummary, Box, Typography } from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';

const CustomAccordionSummary = ({
  unitPlural,
  nr,
  hasError,
  title,
}: {
  unitPlural?: string;
  nr?: number;
  hasError?: boolean;
  title?: string;
}) => {
  const text = Number.isNaN(nr ?? NaN) || (nr ?? 0) < 1 ? '' : ` (${nr})`;

  return (
    <AccordionSummary
      sx={{
        paddingLeft: 0,
      }}
      expandIcon={<ExpandMoreIcon />}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
        }}
      >
        {hasError && (
          <ErrorOutlineIcon
            color="error"
            sx={{ mr: 1, position: 'relative', bottom: 1 }}
          />
        )}
        {unitPlural && (
          <Typography
            variant="sectionHeading"
            sx={{
              textTransform: 'capitalize',
            }}
          >{`${unitPlural} ${text}`}</Typography>
        )}

        {title && (
          <Typography variant="h6" sx={{ pl: 3, pt: 1 }}>
            {title}
          </Typography>
        )}
      </Box>
    </AccordionSummary>
  );
};

export default CustomAccordionSummary;
