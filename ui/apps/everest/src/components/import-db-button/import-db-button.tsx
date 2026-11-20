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

import { Button } from '@mui/material';
import { Link } from 'react-router-dom';
import { useImportAvailability } from 'hooks/api/backup-imports';
import { Messages } from './import-db-button.messages';
import { ImportDbButtonProps } from './import-db-button.types';

export const ImportDbButton = ({ sx }: ImportDbButtonProps) => {
  const { canImport, isLoading } = useImportAvailability();

  if (!canImport && !isLoading) {
    return null;
  }

  return (
    <Button
      component={Link}
      to="/databases/import"
      size="small"
      variant="text"
      disabled={isLoading}
      sx={sx}
      data-testid="import-db-button"
    >
      {Messages.importDatabase}
    </Button>
  );
};
