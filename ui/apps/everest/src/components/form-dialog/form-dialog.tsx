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

import { zodResolver } from '@hookform/resolvers/zod';
import {
  Button,
  DialogActions,
  DialogContent,
  FormGroup,
  ModalProps,
  Typography,
} from '@mui/material';
import { DialogTitle, Dialog } from '@percona/ui-lib';
import { kebabize } from '@percona/utils';
import { useActiveBreakpoint } from 'hooks/utils/useActiveBreakpoint';
import { useMemo } from 'react';
import {
  FieldValues,
  FormProvider,
  SubmitHandler,
  useForm,
} from 'react-hook-form';
import { FormDialogProps } from './form-dialog.types';
import { useCelValidation } from 'components/ui-generator/hooks/use-cel-validation';

export const FormDialog = <T extends FieldValues>({
  isOpen,
  closeModal,
  headerMessage,
  children,
  schema,
  celDependencyGroups = [],
  defaultValues,
  values,
  disableSubmit = false,
  onSubmit,
  cancelMessage = 'Cancel',
  submitMessage = 'Create',
  validationMode = 'onChange',
  subHead2,
  size = 'L',
  submitting = false,
  dataTestId,
  scroll = 'body',
}: FormDialogProps<T>) => {
  const methods = useForm<T>({
    mode: validationMode,
    resolver: zodResolver(schema),
    defaultValues,
    values,
  });
  const {
    formState: { isDirty, isValid },
  } = methods;
  useCelValidation(
    celDependencyGroups,
    methods.control as never,
    methods.trigger as never
  );
  const { isMobile } = useActiveBreakpoint();
  const modalWidth = useMemo(() => {
    if (isMobile) {
      return '80%';
    }

    switch (size) {
      case 'L':
        return '480px';
      case 'XL':
        return '640px';
      case 'XXL':
        return '700px';
      case 'XXXL':
        return '800px';
      default:
        return '640px';
    }
  }, [size, isMobile]);

  const handleSubmit: SubmitHandler<T> = (data) => {
    onSubmit(data);
  };

  const handleClose: ModalProps['onClose'] = (_e, reason) => {
    if (reason === 'backdropClick') {
      if (!isDirty) {
        closeModal();
      }
    } else {
      closeModal();
    }
  };

  return (
    <Dialog
      PaperProps={{ sx: { minWidth: modalWidth } }}
      open={isOpen}
      onClose={handleClose}
      data-testid={dataTestId ? `${dataTestId}-form-dialog` : 'form-dialog'}
      scroll={scroll}
      loading={submitting}
    >
      <DialogTitle onClose={closeModal}>{headerMessage}</DialogTitle>
      <DialogContent>
        {subHead2 && <Typography variant="subHead2">{subHead2}</Typography>}
        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit(handleSubmit)}>
            <FormGroup>
              {typeof children === 'function' ? children(methods) : children}
            </FormGroup>
          </form>
        </FormProvider>
      </DialogContent>
      <DialogActions>
        <Button
          disabled={submitting}
          onClick={closeModal}
          data-testid={`form-dialog-${kebabize(cancelMessage)}`}
        >
          {cancelMessage}
        </Button>
        <Button
          variant="contained"
          onClick={methods.handleSubmit(handleSubmit)}
          disabled={submitting || !isValid || disableSubmit}
          data-testid={`form-dialog-${kebabize(submitMessage)}`}
        >
          {submitMessage}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
