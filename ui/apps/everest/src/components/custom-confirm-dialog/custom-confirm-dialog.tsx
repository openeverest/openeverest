// everest
// Copyright (C) 2023 Percona LLC
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

import { FormDialog } from '../form-dialog';
import { FormDialogProps } from '../form-dialog/form-dialog.types';
import { SubmitHandler } from 'react-hook-form';
import { TextInput } from '@percona/ui-lib';
import { kebabize } from '@percona/utils';
import { customConfirmDialogDefaultValues } from './custom-confirm-dialog-consts';
import { IrreversibleAction } from '../irreversible-action';
import { Box, DialogContent, Tooltip } from '@mui/material';
import { ReactNode } from 'react';
import { CustomCheckbox } from 'components/custom-confirm-dialog/custom-checkbox/custom-checkbox';
import {
  CustomConfirmDialogFields,
  CustomConfirmDialogType,
  customConfirmDialogSchema,
} from './custom-confirm-dialog.types';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';

interface CustomConfirmDialogProps<T extends CustomConfirmDialogType>
  extends Omit<
    FormDialogProps<T>,
    'schema' | 'onSubmit' | 'submitMessage' | 'cancelMessage' | 'children'
  > {
  inputLabel?: string;
  inputPlaceholder?: string;
  handleConfirm: (data: CustomConfirmDialogType) => void;
  selectedId?: string;
  cancelMessage?: string;
  submitMessage?: string;
  dialogContent?: ReactNode;
  confirmationInput?: boolean;
  alertTitle?: string;
  alertMessage: string;
  checkboxMessage?: string;
  disableCheckbox?: boolean;
  hideCheckbox?: boolean;
  tooltipText?: string;
}
export const CustomConfirmDialog = ({
  inputLabel,
  inputPlaceholder,
  handleConfirm,
  cancelMessage = 'Cancel',
  submitMessage = 'Delete',
  confirmationInput = true,
  selectedId = '',
  dialogContent,
  submitting,
  alertTitle = 'Irreversible action',
  alertMessage,
  checkboxMessage,
  disableCheckbox = false,
  hideCheckbox = false,
  tooltipText = '',
  ...props
}: CustomConfirmDialogProps<CustomConfirmDialogType>) => {
  const onSubmit: SubmitHandler<CustomConfirmDialogType> = (data) => {
    handleConfirm(data);
  };

  return (
    <FormDialog
      onSubmit={onSubmit}
      schema={customConfirmDialogSchema(selectedId, confirmationInput)}
      defaultValues={customConfirmDialogDefaultValues(disableCheckbox)}
      dataTestId={selectedId && kebabize(selectedId)}
      submitMessage={submitMessage}
      cancelMessage={cancelMessage}
      submitting={submitting}
      {...props}
    >
      <IrreversibleAction alertTitle={alertTitle} alertMessage={alertMessage} />
      <DialogContent sx={{ px: 1 }}>{dialogContent}</DialogContent>
      {confirmationInput && (
        <TextInput
          name={CustomConfirmDialogFields.confirmInput}
          label={inputLabel}
          labelProps={{ sx: { mt: 0 } }}
          textFieldProps={{
            placeholder: inputPlaceholder,
          }}
        />
      )}
      {checkboxMessage && !hideCheckbox && (
        <CustomCheckbox
          checkboxMessage={
            disableCheckbox ? (
              <Box
                sx={{
                  display: 'flex',
                  mt: 0,
                }}
              >
                {checkboxMessage}
                <Tooltip
                  title={tooltipText}
                  arrow
                  placement="right"
                  sx={{ ml: 1 }}
                >
                  <InfoOutlinedIcon />
                </Tooltip>
              </Box>
            ) : (
              <>{checkboxMessage}</>
            )
          }
          formControlLabelProps={{ sx: { mt: 2, pl: 1 } }}
          disabled={disableCheckbox}
        />
      )}
    </FormDialog>
  );
};
