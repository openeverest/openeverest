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

import React from 'react';
import { Typography, Box } from '@mui/material';
import RoundedBox from 'components/rounded-box';
import { useFormContext, useWatch } from 'react-hook-form';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';

const Header = ({
  title,
  controlComponent,
}: {
  title: string;
  controlComponent?: React.ReactNode;
}) => (
  <Box
    sx={{
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      mb: 1,
    }}
  >
    <Typography variant="sectionHeading">{title}</Typography>
    {controlComponent && (
      <Box
        sx={{
          maxWidth: '40%',
          textAlign: 'right',
        }}
      >
        {controlComponent}
      </Box>
    )}
  </Box>
);

type FormCardProps = {
  title: string;
  controlComponent?: React.ReactNode;
  description?: string;
  cardContent?: React.ReactNode;
};

const FormCard: React.FC<FormCardProps> = ({
  title,
  description = '',
  cardContent,
  controlComponent,
}) => {
  return (
    <RoundedBox
      title={<Header title={title} controlComponent={controlComponent} />}
    >
      {description && <Typography variant="caption">{description}</Typography>}
      {cardContent && <Box sx={{ mt: 1 }}>{cardContent}</Box>}
    </RoundedBox>
  );
};

type FormCardWithCheckProps = {
  title: string;
  controlComponent: React.JSX.Element;
};

const FormCardWithCheck: React.FC<FormCardWithCheckProps> = ({
  title,
  controlComponent,
}) => {
  const { getValues } = useFormContext();

  const fieldValue = getValues(controlComponent!.props?.name);
  return (
    <Box
      className="percona-rounded-box"
      sx={(theme) => ({
        p: 2,
        borderStyle: 'solid',
        borderWidth: '1px',
        borderColor: theme.palette.divider,
        borderRadius: 2,
        display: 'flex',
        justifyContent: 'space-between',
      })}
    >
      <Box sx={{ display: 'flex', alignItems: 'center' }}>
        {fieldValue && <CheckCircleIcon color="success" />}

        <Typography
          sx={{ marginLeft: '5px', marginTop: '5px' }}
          variant="sectionHeading"
        >
          {title}
        </Typography>
      </Box>
      <Box>{controlComponent}</Box>
    </Box>
  );
};

type FormCardWithDialogProps = {
  title: string;
  content: React.ReactNode;
  optional?: boolean;
  sectionSavedKey: string;
};

const FormCardWithDialog: React.FC<FormCardWithDialogProps> = ({
  title,
  content,
  optional = false,
  sectionSavedKey,
}) => {
  const { control } = useFormContext();
  const isSectionSaved = useWatch({ control, name: sectionSavedKey });
  return (
    <Box
      className="percona-rounded-box"
      sx={(theme) => ({
        marginTop: 1,
        p: 2,
        borderStyle: 'solid',
        borderWidth: '1px',
        borderColor: theme.palette.divider,
        borderRadius: 2,
        display: 'flex',
        justifyContent: 'space-between',
      })}
    >
      <Box sx={{ display: 'flex', alignItems: 'center' }}>
        {isSectionSaved && <CheckCircleIcon color="success" />}
        <Typography variant="sectionHeading" sx={{ padding: '6px 8px' }}>
          {title}{' '}
          {optional && <Typography variant="caption">(optional)</Typography>}
        </Typography>
      </Box>
      {content}
    </Box>
  );
};

export { FormCard, FormCardWithCheck, FormCardWithDialog };
