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
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Typography,
} from '@mui/material';
import { readGroupDirection } from 'components/ui-generator/utils/group-params/read-group-direction';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import StackWrapper from '../stack-wrapper';
import { AccordionWrapperProps } from './accordion-wrapper.types';

// TODO can be customizable with props
const AccordionWrapper = ({
  children,
  label,
  groupParams,
}: AccordionWrapperProps) => {
  const direction = readGroupDirection(groupParams);

  return (
    <Accordion defaultExpanded>
      {label && (
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
            }}
          >
            {/* {hasError && (
                            <ErrorOutlineIcon
                                color="error"
                                sx={{ mr: 1, position: 'relative', bottom: 1 }}
                            />
                        )} */}
            <Typography
              variant="sectionHeading"
              sx={{
                textTransform: 'capitalize',
              }}
            >
              {label}
            </Typography>
          </Box>
        </AccordionSummary>
      )}
      <AccordionDetails>
        <StackWrapper direction={direction}>{children}</StackWrapper>
      </AccordionDetails>
    </Accordion>
  );
};
export default AccordionWrapper;
