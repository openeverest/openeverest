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

import { isPlainObject } from 'components/ui-generator/utils/object-path';
import {
  getSectionFormDefaults,
  postprocessSectionForm,
  SectionForm,
  toSectionForm,
} from 'components/ui-generator/utils/section-form';
import {
  CredentialsSecretWrite,
  SecretAlreadyExistsError,
  SecretChangedMeanwhileError,
} from 'hooks/api/secrets';
import {
  ClassContract,
  getCredentialsSecretLabels,
  getCredentialsSection,
  isSecretBackedAuthMode,
} from 'utils/class-contract';
import { toSecretStringData } from 'utils/secrets';
import { CREDENTIALS_SECTION_KEY } from './class-authentication.constants';
import { Messages } from './class-authentication.messages';
import {
  AuthenticationFormValues,
  AuthenticationNouns,
  CredentialsSource,
  SavedAuthentication,
  SecretNamespacePermissions,
  SecretNamingSpec,
} from './class-authentication.types';

// The form the user types the mode's credentials in. New credentials are
// always typed in full, also when editing.
export const toCredentialsForm = (
  contract: ClassContract | undefined,
  mode: string
): SectionForm | undefined => {
  const section = contract && getCredentialsSection(contract, mode);
  return section && toSectionForm(CREDENTIALS_SECTION_KEY, section);
};

// The Secret the object keeps when the user doesn't change the credentials:
// only its current one, and only for the mode it was made for.
export const getKeptCredentialsSecretName = (
  authMode: string,
  saved: SavedAuthentication | undefined
): string | undefined => {
  const secretName = saved?.credentialsSecretRef?.name;
  return secretName && authMode === saved?.mode ? secretName : undefined;
};

// The credentials sources the user may pick for the mode: the object's own
// Secret as it is or updated, else a new one; and an existing Secret of the
// class and mode. None for a mode without a Secret.
export const getCredentialsSources = (
  {
    authMode,
    namespace,
  }: Pick<AuthenticationFormValues, 'authMode'> & { namespace: string },
  saved: SavedAuthentication | undefined,
  { canCreate, canRead, canUpdate }: SecretNamespacePermissions
): CredentialsSource[] => {
  if (!authMode || !isSecretBackedAuthMode(authMode)) {
    return [];
  }
  const may = (namespaces: string[]) => namespaces.includes(namespace);
  const ownSources = getKeptCredentialsSecretName(authMode, saved)
    ? [
        CredentialsSource.Keep,
        // Updating reads the Secret's version first.
        ...(may(canUpdate) && may(canRead) ? [CredentialsSource.Update] : []),
      ]
    : may(canCreate)
      ? [CredentialsSource.New]
      : [];
  return may(canRead)
    ? [...ownSources, CredentialsSource.Existing]
    : ownSources;
};

// The picked source, or the first offered once the pick no longer applies
// (e.g. after the mode changed).
export const getCredentialsSource = (
  values: Pick<AuthenticationFormValues, 'authMode' | 'credentialsSource'> & {
    namespace: string;
  },
  saved: SavedAuthentication | undefined,
  permissions: SecretNamespacePermissions
): CredentialsSource | undefined => {
  const sources = getCredentialsSources(values, saved, permissions);
  return sources.includes(values.credentialsSource)
    ? values.credentialsSource
    : sources[0];
};

// Whether the form takes new credentials for the source.
export const takesCredentials = (source: CredentialsSource | undefined) =>
  source === CredentialsSource.New || source === CredentialsSource.Update;

// The authentication fields for a saved object or, without one, a new one.
export const getAuthenticationFormValues = (
  saved: SavedAuthentication | undefined,
  credentialsForm?: SectionForm
): AuthenticationFormValues => {
  const authMode = saved?.mode ?? '';
  return {
    authMode,
    credentialsSource: getKeptCredentialsSecretName(authMode, saved)
      ? CredentialsSource.Keep
      : CredentialsSource.New,
    credentialsSecretName: '',
    existingSecretName: '',
    credentials: credentialsForm ? getSectionFormDefaults(credentialsForm) : {},
  };
};

const toCredentialsSecretWrite = ({
  values,
  contract,
  credentialsForm,
  source,
  keptSecretName,
  defaultSecretName,
}: {
  values: AuthenticationFormValues;
  contract: ClassContract | undefined;
  credentialsForm?: SectionForm;
  source: CredentialsSource | undefined;
  keptSecretName: string | undefined;
  defaultSecretName: string;
}): CredentialsSecretWrite | undefined => {
  const stringData = () =>
    toSecretStringData(
      credentialsForm
        ? postprocessSectionForm(credentialsForm, values.credentials)
        : {}
    );
  if (!contract) {
    return undefined;
  }
  const labels = getCredentialsSecretLabels(contract, values.authMode);
  if (source === CredentialsSource.New) {
    return {
      action: 'create',
      name: values.credentialsSecretName || defaultSecretName,
      labels,
      stringData: stringData(),
    };
  }
  if (source === CredentialsSource.Update && keptSecretName) {
    return {
      action: 'update',
      name: keptSecretName,
      labels,
      stringData: stringData(),
    };
  }
  return undefined;
};

// What a save sends for the authentication: the reference the object gets,
// and the Secret written before it, if any. Credentials go to the Secret
// only; the object merely references it.
export const getAuthenticationRequest = ({
  values,
  contract,
  credentialsForm,
  saved,
  permissions,
  defaultSecretName,
}: {
  values: AuthenticationFormValues & { namespace: string };
  contract: ClassContract | undefined;
  credentialsForm?: SectionForm;
  saved: SavedAuthentication | undefined;
  permissions: SecretNamespacePermissions;
  defaultSecretName: string;
}): {
  authentication: SavedAuthentication;
  credentialsSecret?: CredentialsSecretWrite;
} => {
  const source = getCredentialsSource(values, saved, permissions);
  const keptSecretName = getKeptCredentialsSecretName(values.authMode, saved);
  const credentialsSecret = toCredentialsSecretWrite({
    values,
    contract,
    credentialsForm,
    source,
    keptSecretName,
    defaultSecretName,
  });
  const secretName =
    source === CredentialsSource.Existing
      ? values.existingSecretName
      : (credentialsSecret?.name ?? keptSecretName);

  return {
    authentication: {
      mode: values.authMode,
      ...(source &&
        secretName && { credentialsSecretRef: { name: secretName } }),
    },
    credentialsSecret,
  };
};

export const getCredentialsSecretErrorMessage = (
  error: SecretAlreadyExistsError | SecretChangedMeanwhileError,
  nouns: AuthenticationNouns
) => {
  if (error instanceof SecretChangedMeanwhileError) {
    return Messages.secretChangedMeanwhile(error.secretName);
  }
  return error.reusable
    ? Messages.secretExists(error.secretName)
    : Messages.secretTaken(error.secretName, nouns.class);
};

const toCanonicalJson = (value: unknown): unknown => {
  if (Array.isArray(value)) {
    return value.map(toCanonicalJson);
  }
  if (!isPlainObject(value)) {
    return value;
  }
  return Object.fromEntries(
    Object.keys(value)
      .sort()
      .map((key) => [key, toCanonicalJson(value[key])])
  );
};

// As the server compares a stored and an updated spec: as JSON documents.
const isSameDocument = (a: unknown, b: unknown) =>
  JSON.stringify(toCanonicalJson(a)) === JSON.stringify(toCanonicalJson(b));

// Why the save can't go ahead, when it needs use on Secrets the user may not
// use: the server requires it on every Secret an object names when the
// object is created or its spec changes, Secret by Secret, as checked here.
// Run before anything is written, so a refused save leaves no Secret changed.
export const getSecretUseProblem = async ({
  spec,
  savedSpec,
  namespace,
  canUseSecret,
}: {
  spec: SecretNamingSpec;
  savedSpec: SecretNamingSpec | undefined;
  namespace: string;
  canUseSecret: (name: string) => Promise<boolean>;
}): Promise<string | undefined> => {
  const names = [
    ...new Set(
      [
        spec.authentication?.credentialsSecretRef?.name,
        spec.tls?.ca?.secretKeyRef?.name,
      ].filter((name): name is string => !!name)
    ),
  ];
  if (names.length === 0 || (savedSpec && isSameDocument(spec, savedSpec))) {
    return undefined;
  }
  const usable = await Promise.all(names.map(canUseSecret));
  const denied = names.filter((_, index) => !usable[index]);
  return denied.length > 0
    ? Messages.secretUseRequired(denied, namespace)
    : undefined;
};
