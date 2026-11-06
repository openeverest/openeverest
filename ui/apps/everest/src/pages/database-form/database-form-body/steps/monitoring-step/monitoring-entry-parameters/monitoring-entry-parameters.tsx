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

import { Box, Typography } from '@mui/material';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { MonitoringSectionForm } from 'utils/monitoring-classes';
import { MONITORING_DESTINATIONS_FIELD } from '../monitoring-step.constants';
import { Messages } from '../monitoring-step.messages';

interface MonitoringEntryParametersProps {
  destinationName: string;
  entryIndex: number;
  form: MonitoringSectionForm;
  namespace: string;
}

// The class's instance form, mounted at the entry so its relative paths
// (e.g. parameters.resources.requests.cpu) stay within that entry.
export const MonitoringEntryParameters = ({
  destinationName,
  entryIndex,
  form,
  namespace,
}: MonitoringEntryParametersProps) => (
  <Box data-testid={`monitoring-entry-parameters-${destinationName}`}>
    <Typography variant="subtitle2">
      {Messages.entryParameters(destinationName)}
    </Typography>
    <UIGenerator
      sectionKey={form.sectionKey}
      sections={form.sections}
      root={`${MONITORING_DESTINATIONS_FIELD}.${entryIndex}`}
      namespace={namespace}
      formMode={FormMode.New}
    />
  </Box>
);
