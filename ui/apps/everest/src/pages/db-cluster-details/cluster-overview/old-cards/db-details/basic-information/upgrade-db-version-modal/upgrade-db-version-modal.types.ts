// everest
// Copyright (C) 2023 Percona LLC
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

import { z } from 'zod';
import { DbEngine } from 'shared-types/dbEngines.types';
import { dbVersionSchema } from 'components/cluster-form/db-version/db-version-schema';

export interface UpgradeModalProps {
  open: boolean;
  handleCloseModal: () => void;
  handleSubmitModal: (dbVersion: string) => void;
  dbVersionsUpgradeList: DbEngine;
  version: string;
  submitting: boolean;
}

export type UpgradeModalFormType = z.infer<typeof dbVersionSchema>;
