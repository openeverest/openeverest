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
import { KeyValueArray, TextInput } from '@percona/ui-lib';
import {
  OpenApiParameterKind,
  OpenApiParametersFieldsProps,
} from './openapi-parameters-fields.types';
import { Messages } from './openapi-parameters-fields.messages';

// A small form for an OpenAPI parameters schema that ships no uiSchema.
export const OpenApiParametersFields = ({
  name,
  parameterFields: { fields, skipped },
}: OpenApiParametersFieldsProps) => (
  <>
    {fields.map((field) =>
      field.kind === OpenApiParameterKind.Text ? (
        <TextInput
          key={field.name}
          name={`${name}.${field.name}`}
          label={field.name}
          isRequired={field.required}
          textFieldProps={{ helperText: field.description }}
        />
      ) : (
        <KeyValueArray
          key={field.name}
          name={`${name}.${field.name}`}
          label={field.name}
        />
      )
    )}
    {skipped.length > 0 && (
      <Typography variant="helperText" color="text.secondary" sx={{ mt: 2 }}>
        {Messages.skipped(skipped)}
      </Typography>
    )}
  </>
);
