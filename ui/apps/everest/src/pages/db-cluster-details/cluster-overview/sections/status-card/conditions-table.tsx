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
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import { format } from 'date-fns';
import { DATE_FORMAT } from 'consts';
import { InstanceCondition } from 'shared-types/instance.types';
import { Messages } from './status-card.messages';

interface ConditionsTableProps {
  conditions: InstanceCondition[];
}

// Every condition exactly as the Instance reports it, including the types
// this UI has no title for. The rest of the card is a reading of this table.
export const ConditionsTable = ({ conditions }: ConditionsTableProps) => {
  if (conditions.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
        {Messages.noConditions}
      </Typography>
    );
  }

  return (
    <TableContainer sx={{ mt: 1 }}>
      <Table size="small" data-testid="status-conditions-table">
        <TableHead>
          <TableRow>
            <TableCell>{Messages.conditions.type}</TableCell>
            <TableCell>{Messages.conditions.status}</TableCell>
            <TableCell>{Messages.conditions.reason}</TableCell>
            <TableCell>{Messages.conditions.message}</TableCell>
            <TableCell>{Messages.conditions.lastTransition}</TableCell>
            <TableCell align="right">
              {Messages.conditions.observedGeneration}
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {conditions.map((condition) => (
            <TableRow
              key={condition.type}
              data-testid={`status-condition-${condition.type}`}
            >
              <TableCell sx={{ whiteSpace: 'nowrap' }}>
                {condition.type}
              </TableCell>
              <TableCell>{condition.status}</TableCell>
              <TableCell>{condition.reason}</TableCell>
              <TableCell
                sx={{
                  minWidth: 200,
                  wordBreak: 'break-word',
                  whiteSpace: 'pre-line',
                }}
              >
                {condition.message}
              </TableCell>
              <TableCell sx={{ whiteSpace: 'nowrap' }}>
                {format(new Date(condition.lastTransitionTime), DATE_FORMAT)}
              </TableCell>
              <TableCell align="right">
                {condition.observedGeneration ?? '\u2014'}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};
