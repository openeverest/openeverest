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
import RoundedBox from 'components/rounded-box';
import { BorderedWrapperProps } from './bordered-wrapper.types';

// A static bordered card grouping distinct fields. The heading region renders
// only when a label and/or description is given; with neither it's a plain
// bordered box. Shared card chrome (RoundedBox) so collapsible/toggleable cards
// stay pixel-identical to a static bordered card.
const BorderedWrapper = ({
  label,
  description,
  children,
}: BorderedWrapperProps) => {
  const heading =
    label || description ? (
      <Stack sx={{ gap: 0.5, mb: 1 }}>
        {label && <Typography variant="sectionHeading">{label}</Typography>}
        {description && (
          <Typography variant="caption" sx={{ color: 'text.secondary' }}>
            {description}
          </Typography>
        )}
      </Stack>
    ) : undefined;

  return <RoundedBox title={heading}>{children}</RoundedBox>;
};

export default BorderedWrapper;
