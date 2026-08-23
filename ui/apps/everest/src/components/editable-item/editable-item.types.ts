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

import { IconButtonProps, PaperProps } from '@mui/material';

export interface EditableItemProps {
  children: React.ReactNode;
  editButtonProps?: IconButtonProps;
  deleteButtonProps?: IconButtonProps & {
    tooltipMessage?: string;
  };
  dataTestId: string;
  paperProps?: PaperProps;
  endText?: string;
  // Optional control (e.g. a toggle) rendered at the end of the right actions cluster.
  controls?: React.ReactNode;
}
