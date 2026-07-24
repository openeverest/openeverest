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
  Typography,
  CardProps as MuiCardProps,
  CardContent,
  Box,
  Button,
  CardActions,
  ButtonProps,
  TypographyProps,
  CardContentProps,
  BoxProps,
  CardActionsProps,
} from '@mui/material';
import { kebabize } from '@percona/utils';

export interface CardProps extends Omit<MuiCardProps, 'content'> {
  content: ReactNode;
  dataTestId: string;
  cardActions?: ActionProps[];
  cardActionsProps?: CardActionsProps;
  headerProps?: TypographyProps;
  cardContentProps?: CardContentProps;
  contentWrapperProps?: BoxProps;
}

export interface ActionProps extends ButtonProps {
  text: string;
}

const Card = ({
  title,
  content,
  sx,
  cardActions,
  headerProps,
  cardContentProps,
  contentWrapperProps,
  cardActionsProps,
  dataTestId,
  ...props
}: CardProps) => {
  return (
    <MuiCard
      data-testid={`${dataTestId}-card`}
      sx={[
        { width: '320px', height: 'fit-content' },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
      {...props}
    >
      <CardContent
        data-testid={`${dataTestId}-card-content`}
        {...cardContentProps}
        sx={{
          '&:last-child': {
            p: 2,
          },
          ...cardContentProps?.sx,
        }}
      >
        {title && (
          <Typography
            data-testid={`${dataTestId}-card-header`}
            variant="h5"
            {...headerProps}
            sx={{ mb: 4, ...headerProps?.sx }}
          >
            {title}
          </Typography>
        )}
        <Box
          data-testid={`${dataTestId}-card-content-wrapper`}
          {...contentWrapperProps}
        >
          {content}
        </Box>
        {cardActions && (
          <CardActions
            data-testid={`${dataTestId}-card-actions`}
            {...cardActionsProps}
            sx={{ p: 0, mt: 4, ...cardActionsProps?.sx }}
          >
            {cardActions.map(({ text, ...buttonProps }) => (
              <Button
                key={text}
                data-testid={`${kebabize(text)}-button`}
                {...buttonProps}
              >
                {text}
              </Button>
            ))}
          </CardActions>
        )}
      </CardContent>
    </MuiCard>
  );
};

export default Card;
