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

import { SectionForm } from 'components/ui-generator/utils/section-form';
import { ClassContract } from 'utils/class-contract';

// Where the object's credentials come from: its current Secret as it is or
// with new credentials, a new Secret, or an existing one of the class.
export enum CredentialsSource {
  Keep = 'keep',
  Update = 'update',
  New = 'new',
  Existing = 'existing',
}

export interface AuthenticationFormValues {
  authMode: string;
  credentialsSource: CredentialsSource;
  credentialsSecretName: string;
  existingSecretName: string;
  // The Secret's keys, as the class's credentials form fills them.
  credentials: Record<string, unknown>;
}

// spec.authentication of a saved object.
export interface SavedAuthentication {
  mode: string;
  credentialsSecretRef?: { name: string };
}

// The parts of an object's spec that name Secrets: its credentials and its
// CA bundle.
export interface SecretNamingSpec {
  authentication?: SavedAuthentication;
  tls?: { ca?: { secretKeyRef?: { name: string } } };
}

// The namespaces the user may create, read and update Secrets in.
export interface SecretNamespacePermissions {
  canCreate: string[];
  canRead: string[];
  canUpdate: string[];
}

// What the messages call the object the block is part of and its class,
// e.g. "destination" and "monitoring class".
export interface AuthenticationNouns {
  object: string;
  class: string;
}

export interface CredentialsFieldsProps {
  contract: ClassContract;
  // The class's credentials form for the picked mode, if it keeps them in a
  // Secret.
  form?: SectionForm;
  saved?: SavedAuthentication;
  namespace: string;
  // The name a new Secret gets when the user doesn't enter one.
  defaultSecretName: string;
  nouns: AuthenticationNouns;
  secretPermissions: SecretNamespacePermissions;
  secretPermissionsLoading: boolean;
}
