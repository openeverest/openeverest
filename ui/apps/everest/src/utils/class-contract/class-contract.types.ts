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

import { Section } from 'components/ui-generator/ui-generator.types';
import { SecretListFilter } from 'shared-types/api.types';
import { AuthModeCredentials } from 'utils/auth-mode-credentials';

// What a class publishes for the forms of the objects that use it, whatever
// its kind. Each kind's adapter (e.g. toMonitoringClassContract) is the only
// code that reads the kind's CRD.
export interface ClassContract {
  name: string;
  title: string;
  description?: string;
  authModes: AuthModeCredentials[];
  // The class's own credentials forms, keyed by auth mode.
  credentialsForms: Partial<Record<string, Section>>;
  // The label the Secrets API checks a credentials Secret of the class by,
  // and the list filter that selects those Secrets.
  credentialsSecretLabel: string;
  credentialsSecretFilter: SecretListFilter;
}
