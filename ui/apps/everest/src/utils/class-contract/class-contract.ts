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

import { SECRET_DEFINITION_LABEL } from 'components/ui-generator/secret-field/secret-create-modal/secret-create-modal.constants';
import { Section } from 'components/ui-generator/ui-generator.types';
import { SecretListFilter } from 'shared-types/api.types';
import { getAuthModeCredentialsSection } from 'utils/auth-mode-credentials';
import { ClassContract } from './class-contract.types';

// Modes core resolves without a Secret.
const SECRETLESS_AUTH_MODES = ['Anonymous', 'WorkloadIdentity'];

export const isSecretBackedAuthMode = (mode: string) =>
  !SECRETLESS_AUTH_MODES.includes(mode);

// The credentials form of a mode that keeps them in a Secret: the class's own
// or, when it ships none, one masked input per credential key the mode
// declares.
export const getCredentialsSection = (
  contract: ClassContract,
  mode: string
): Section | undefined =>
  mode && isSecretBackedAuthMode(mode)
    ? getAuthModeCredentialsSection(
        contract.authModes,
        contract.credentialsForms,
        mode
      )
    : undefined;

// The labels the Secrets API checks a credentials Secret against: the class
// that defines it and the authentication mode whose keys it holds.
export const getCredentialsSecretLabels = (
  contract: ClassContract,
  mode: string
): Record<string, string> => ({
  [contract.credentialsSecretLabel]: contract.name,
  [SECRET_DEFINITION_LABEL]: mode,
});

// Selects the Secrets holding credentials of the class and mode.
export const getCredentialsSecretFilter = (
  contract: ClassContract,
  mode: string
): SecretListFilter => ({
  ...contract.credentialsSecretFilter,
  definition: mode,
});
