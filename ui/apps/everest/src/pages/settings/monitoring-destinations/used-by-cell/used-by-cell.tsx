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

import { List, ListItem } from '@mui/material';
import { ShowMoreDialog } from '@percona/ui-lib';
import { Messages } from '../monitoring-destinations.messages';
import { DestinationUsage } from '../monitoring-destinations.types';

interface UsedByCellProps {
  destination: string;
  usage: DestinationUsage;
}

// A count that opens the full list, so the cell stays one line however
// many instances use the destination.
export const UsedByCell = ({
  destination,
  usage: { count, instances },
}: UsedByCellProps) => {
  if (count === 0) {
    return Messages.none;
  }
  const hidden = count - instances.length;
  return (
    <ShowMoreDialog
      dialogTitle={Messages.usedByDialogTitle(destination)}
      linkLabel={Messages.usedByCount(count)}
      linkTypographyProps={{ variant: 'body2' }}
      dataTestId={`used-by-${destination}`}
      dialogProps={{ maxWidth: 'sm' }}
    >
      <List dense>
        {[...instances].sort().map((instance) => (
          <ListItem key={instance} disableGutters>
            {instance}
          </ListItem>
        ))}
        {hidden > 0 && (
          <ListItem disableGutters>{Messages.hiddenInstances(hidden)}</ListItem>
        )}
      </List>
    </ShowMoreDialog>
  );
};
