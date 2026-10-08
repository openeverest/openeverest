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

import { ReactNode, useState } from 'react';
import { Box, Tab, Tabs } from '@mui/material';
import { Messages } from '../pod-scheduling-policy-section.messages';

interface SchedulingPolicyTabsProps {
  // One tab per component, with the number of rule groups it carries.
  tabs: { key: string; count: number }[];
  children: (activeKey: string) => ReactNode;
}

export const SchedulingPolicyTabs = ({
  tabs,
  children,
}: SchedulingPolicyTabsProps) => {
  // Open on the first component that has rules; chosen once so deleting the
  // last rule doesn't jump the user to another tab.
  const [selected, setSelected] = useState(
    () => tabs.find(({ count }) => count > 0)?.key
  );
  const keys = tabs.map(({ key }) => key);
  const activeKey = selected && keys.includes(selected) ? selected : keys[0];

  return (
    <Box>
      <Tabs
        value={activeKey}
        onChange={(_, value) => setSelected(value)}
        variant="compact"
        sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}
      >
        {tabs.map(({ key, count }) => (
          <Tab
            key={key}
            value={key}
            label={Messages.tabLabel(key, count)}
            data-testid={`scheduling-component-tab-${key}`}
          />
        ))}
      </Tabs>
      {children(activeKey)}
    </Box>
  );
};
