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

import { Stack, Typography } from '@mui/material';
import { AffinityRule, AffinityType } from 'shared-types/affinity.types';

const showRuleProperty = (prop: string | undefined) => {
  return prop ? ` | ${prop}` : '';
};

export const AffinityItem = ({ rule }: { rule: AffinityRule }) => {
  const valuesToShow =
    rule.type === AffinityType.NodeAffinity ? [] : [rule.topologyKey];
  return (
    <Stack
      direction="row"
      sx={{
        alignItems: 'center',
        width: '100%',
      }}
    >
      <Stack
        sx={{
          width: '50%',
        }}
      >
        <Typography variant="body1">
          {rule.type}
          {[...valuesToShow, rule.key, rule.operator, rule.values].map((prop) =>
            showRuleProperty(prop)
          )}
        </Typography>
      </Stack>
    </Stack>
  );
};
