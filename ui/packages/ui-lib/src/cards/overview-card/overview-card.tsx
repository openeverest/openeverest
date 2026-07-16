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

import { ReactNode } from 'react';
import {
  Card as MuiCard,
  CardProps as MuiCardProps,
  CardContent,
  CardContentProps,
  CardHeader,
  CardHeaderProps,
} from '@mui/material';
import { DatabaseIcon } from '../../icons';

export interface OverviewCardProps extends Omit<MuiCardProps, 'content'> {
  cardHeaderProps?: CardHeaderProps;
  cardContentProps?: CardContentProps;
  dataTestId: string;
  children: ReactNode;
}

const OverviewCard = ({
  cardHeaderProps,
  children,
  sx,
  cardContentProps,
  dataTestId,
  ...props
}: OverviewCardProps) => {
  return (
    <MuiCard
      variant="grey"
      sx={[
        { width: '368px', height: 'fit-content' },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
      data-testid={dataTestId}
      {...props}
    >
      {cardHeaderProps?.title && (
        <CardHeader
          data-testid={`${dataTestId}-card-header`}
          title={cardHeaderProps?.title}
          avatar={<DatabaseIcon />}
          {...cardHeaderProps}
        />
      )}
      <CardContent {...cardContentProps}>{children}</CardContent>
    </MuiCard>
  );
};

export default OverviewCard;
