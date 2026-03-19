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

import {
  PostprocessOptions,
  postprocessSchemaData,
} from 'components/ui-generator/utils/postprocess/postprocess-schema';

// Deprecated wrapper kept for backward compatibility with existing imports.
// Prefer using postprocessSchemaData directly from ui-generator utils.
export const formSubmitPostProcessing = (
  _baseValues: Record<string, unknown>,
  submittedFormValues: Record<string, unknown>,
  options?: PostprocessOptions
): Record<string, unknown> => {
  return postprocessSchemaData(submittedFormValues, options);
};
