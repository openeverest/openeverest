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

import { Typography } from '@mui/material';
import { AffinityGroup } from '../affinity-group.types';
import { matchesAllPods, matchesNoPods } from '../affinity-group-converter';
import { Messages } from '../affinity-rule-editor.messages';

// The effect of an absent or empty label selector, which no condition shows.
export const GroupNotes = ({ group }: { group: AffinityGroup }) => {
  if (matchesNoPods(group)) {
    return (
      <Typography variant="caption" sx={{ color: 'warning.main' }}>
        {Messages.noLabelSelector}
      </Typography>
    );
  }
  if (matchesAllPods(group)) {
    return (
      <Typography variant="caption" sx={{ color: 'text.secondary' }}>
        {Messages.emptyLabelSelector}
      </Typography>
    );
  }
  return null;
};
