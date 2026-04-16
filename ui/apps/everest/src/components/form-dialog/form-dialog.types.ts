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

import { DialogProps } from '@mui/material';
import { ReactNode } from 'react';

import {
  DefaultValues,
  FieldValues,
  UseFormReturn,
  ValidationMode,
} from 'react-hook-form';
import { z, ZodTypeDef } from 'zod';

export interface FormDialogProps<T extends FieldValues> {
  isOpen: boolean;
  closeModal: () => void;
  headerMessage: string;
  schema: z.Schema<unknown, ZodTypeDef>;
  celDependencyGroups?: string[][];
  defaultValues?: DefaultValues<T>;
  values?: T;
  onSubmit: (data: T) => void;
  children: ((formMethods: UseFormReturn<T>) => ReactNode) | ReactNode;
  cancelMessage?: string;
  submitMessage?: string;
  validationMode?: keyof ValidationMode;
  size?: 'L' | 'XL' | 'XXL' | 'XXXL';
  subHead2?: string;
  submitting?: boolean;
  disableSubmit?: boolean;
  dataTestId?: string;
  scroll?: DialogProps['scroll'];
}
