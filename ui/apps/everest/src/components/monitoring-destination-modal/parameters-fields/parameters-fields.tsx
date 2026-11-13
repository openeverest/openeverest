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

import { Typography } from '@mui/material';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { DestinationFormFields } from '../monitoring-destination-modal.constants';
import { Messages } from '../monitoring-destination-modal.messages';
import { DestinationClassForms } from '../monitoring-destination-modal.types';

interface ParametersFieldsProps {
  forms: DestinationClassForms;
  namespace: string;
  formMode: FormMode;
}

// The class's destination form or, when it ships none, a form generated from
// its destinationParametersSchema.
export const ParametersFields = ({
  forms: { destination, skippedParameters },
  namespace,
  formMode,
}: ParametersFieldsProps) => (
  <>
    {destination && (
      <UIGenerator
        sectionKey={destination.sectionKey}
        sections={destination.sections}
        root={DestinationFormFields.spec}
        namespace={namespace}
        formMode={formMode}
      />
    )}
    {skippedParameters.length > 0 && (
      <Typography variant="helperText" color="text.secondary" sx={{ mt: 2 }}>
        {Messages.skippedParameters(skippedParameters)}
      </Typography>
    )}
  </>
);
