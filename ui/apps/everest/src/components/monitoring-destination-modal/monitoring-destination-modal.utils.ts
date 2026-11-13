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

import { FormMode } from 'components/ui-generator/ui-generator.types';
import { isPlainObject } from 'components/ui-generator/utils/object-path';
import { sectionFromOpenApiSchema } from 'components/ui-generator/utils/openapi-section';
import {
  CredentialsSecretWrite,
  SaveMonitoringDestinationRequest,
  SecretAlreadyExistsError,
  SecretChangedMeanwhileError,
} from 'hooks/api/monitoring-destinations';
import {
  MonitoringClass,
  MonitoringDestination,
  MonitoringDestinationSpec,
} from 'shared-types/monitoring-destinations.types';
import {
  getCredentialsSection,
  getMonitoringClassName,
  getMonitoringClassUiSchema,
  getSectionFormDefaults,
  getSectionFormValues,
  isSecretBackedAuthMode,
  postprocessSectionForm,
  toMonitoringSectionForm,
} from 'utils/monitoring-classes';
import {
  getCredentialsSecretName,
  getDefaultCredentialsSecretName,
} from 'utils/monitoring-destinations';
import { toSecretStringData } from 'utils/secrets';
import {
  CREDENTIALS_SECTION_KEY,
  DESTINATION_SECTION_KEY,
} from './monitoring-destination-modal.constants';
import { Messages } from './monitoring-destination-modal.messages';
import {
  CredentialsSource,
  DestinationClassForms,
  MonitoringDestinationFormValues,
  SecretNamespacePermissions,
} from './monitoring-destination-modal.types';

export const getDestinationClassForms = (
  monitoringClass: MonitoringClass | undefined,
  authMode: string,
  formMode: FormMode
): DestinationClassForms => {
  if (!monitoringClass) {
    return { skippedParameters: [] };
  }
  const { destination } = getMonitoringClassUiSchema(monitoringClass);
  const credentials =
    authMode && isSecretBackedAuthMode(authMode)
      ? getCredentialsSection(monitoringClass, authMode)
      : undefined;

  // A class without a destination form gets one generated from its
  // destinationParametersSchema.
  const generated = destination
    ? undefined
    : sectionFromOpenApiSchema(
        monitoringClass.spec.destinationParametersSchema?.openAPIV3Schema,
        'parameters'
      );
  const destinationSection = destination ?? generated?.section;

  return {
    destination:
      destinationSection &&
      toMonitoringSectionForm(
        DESTINATION_SECTION_KEY,
        destinationSection,
        formMode
      ),
    skippedParameters: generated?.skippedProperties ?? [],
    // New credentials are always typed in full, also when editing.
    credentials:
      credentials &&
      toMonitoringSectionForm(CREDENTIALS_SECTION_KEY, credentials),
  };
};

export const getFormMode = (original: MonitoringDestination | undefined) =>
  original ? FormMode.Edit : FormMode.New;

// The class-dependent fields of a new destination of the class.
export const getNewDestinationClassDefaults = (
  monitoringClass: MonitoringClass | undefined
): Pick<
  MonitoringDestinationFormValues,
  'authMode' | 'spec' | 'credentials'
> => {
  const authModes = monitoringClass?.spec.authModes ?? [];
  const authMode = authModes.length === 1 ? authModes[0].name : '';
  const forms = getDestinationClassForms(
    monitoringClass,
    authMode,
    FormMode.New
  );
  return {
    authMode,
    spec: forms.destination ? getSectionFormDefaults(forms.destination) : {},
    credentials: forms.credentials
      ? getSectionFormDefaults(forms.credentials)
      : {},
  };
};

// The Secret the destination keeps when the user doesn't change the
// credentials: only its current one, and only for the mode it was made for.
export const getKeptCredentialsSecretName = (
  authMode: string,
  original: MonitoringDestination | undefined
): string | undefined => {
  const secretName = original && getCredentialsSecretName(original);
  return secretName && authMode === original.spec.authentication?.mode
    ? secretName
    : undefined;
};

// The credentials sources the user may pick for the mode: the destination's
// own Secret as it is or updated, else a new one; and an existing Secret of
// the class and mode. None for a mode without a Secret.
export const getCredentialsSources = (
  {
    authMode,
    namespace,
  }: Pick<MonitoringDestinationFormValues, 'authMode' | 'namespace'>,
  original: MonitoringDestination | undefined,
  { canCreate, canRead, canUpdate }: SecretNamespacePermissions
): CredentialsSource[] => {
  if (!authMode || !isSecretBackedAuthMode(authMode)) {
    return [];
  }
  const may = (namespaces: string[]) => namespaces.includes(namespace);
  const ownSources = getKeptCredentialsSecretName(authMode, original)
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
  values: Pick<
    MonitoringDestinationFormValues,
    'authMode' | 'namespace' | 'credentialsSource'
  >,
  original: MonitoringDestination | undefined,
  permissions: SecretNamespacePermissions
): CredentialsSource | undefined => {
  const sources = getCredentialsSources(values, original, permissions);
  return sources.includes(values.credentialsSource)
    ? values.credentialsSource
    : sources[0];
};

// Whether the form takes new credentials for the source.
export const takesCredentials = (source: CredentialsSource | undefined) =>
  source === CredentialsSource.New || source === CredentialsSource.Update;

export const getFormDefaults = ({
  original,
  namespace,
  classes,
  className = '',
}: {
  original?: MonitoringDestination;
  namespace: string;
  classes: MonitoringClass[];
  className?: string;
}): MonitoringDestinationFormValues => {
  const empty: MonitoringDestinationFormValues = {
    name: '',
    namespace,
    className: '',
    authMode: '',
    credentialsSource: CredentialsSource.New,
    credentialsSecretName: '',
    existingSecretName: '',
    credentials: {},
    spec: {},
    insecureSkipVerify: false,
    caSecretName: '',
    caSecretKey: '',
  };
  if (!original) {
    const monitoringClass = classes.find(
      (c) => getMonitoringClassName(c) === className
    );
    return monitoringClass
      ? {
          ...empty,
          className,
          ...getNewDestinationClassDefaults(monitoringClass),
        }
      : empty;
  }

  const originalClassName = original.spec.classRef.name;
  const authMode = original.spec.authentication?.mode ?? '';
  const forms = getDestinationClassForms(
    classes.find((c) => getMonitoringClassName(c) === originalClassName),
    authMode,
    FormMode.Edit
  );
  const parameters: unknown = original.spec.parameters;
  const ca = original.spec.tls?.ca?.secretKeyRef;

  return {
    ...empty,
    name: original.metadata?.name ?? '',
    namespace: original.metadata?.namespace ?? namespace,
    className: originalClassName,
    authMode,
    credentialsSource: getKeptCredentialsSecretName(authMode, original)
      ? CredentialsSource.Keep
      : CredentialsSource.New,
    credentials: forms.credentials
      ? getSectionFormDefaults(forms.credentials)
      : {},
    spec: forms.destination
      ? getSectionFormValues(forms.destination, {
          parameters: isPlainObject(parameters) ? parameters : {},
        })
      : {},
    insecureSkipVerify: !!original.spec.tls?.insecureSkipVerify,
    caSecretName: ca?.name ?? '',
    caSecretKey: ca?.key ?? '',
  };
};

const toParameters = (
  values: MonitoringDestinationFormValues,
  forms: DestinationClassForms,
  original: MonitoringDestination | undefined
): Record<string, unknown> => {
  // Without a form nothing is editable, so the saved parameters are kept.
  if (!forms.destination) {
    const current: unknown = original?.spec.parameters;
    return isPlainObject(current) ? current : {};
  }
  // Saved parameters the form doesn't own ride along in the form values
  // (getSectionFormValues keeps them), so they survive the save too.
  const { parameters } = postprocessSectionForm(values.spec, forms.destination);
  return isPlainObject(parameters) ? parameters : {};
};

const toTls = ({
  insecureSkipVerify,
  caSecretName,
  caSecretKey,
}: MonitoringDestinationFormValues): MonitoringDestinationSpec['tls'] => {
  if (insecureSkipVerify) {
    return { insecureSkipVerify: true };
  }
  return caSecretName && caSecretKey
    ? { ca: { secretKeyRef: { name: caSecretName, key: caSecretKey } } }
    : undefined;
};

const toCredentialsSecretWrite = (
  values: MonitoringDestinationFormValues,
  forms: DestinationClassForms,
  source: CredentialsSource | undefined,
  keptSecretName: string | undefined
): CredentialsSecretWrite | undefined => {
  const stringData = () =>
    toSecretStringData(
      forms.credentials
        ? postprocessSectionForm(values.credentials, forms.credentials)
        : {}
    );
  if (source === CredentialsSource.New) {
    return {
      action: 'create',
      name:
        values.credentialsSecretName ||
        getDefaultCredentialsSecretName(values.name),
      stringData: stringData(),
    };
  }
  if (source === CredentialsSource.Update && keptSecretName) {
    return { action: 'update', name: keptSecretName, stringData: stringData() };
  }
  return undefined;
};

// The requests a submit makes. Credentials go to the Secret only; the
// destination merely references it.
export const buildSaveRequest = ({
  values,
  monitoringClass,
  original,
  secretPermissions,
}: {
  values: MonitoringDestinationFormValues;
  monitoringClass: MonitoringClass | undefined;
  original?: MonitoringDestination;
  secretPermissions: SecretNamespacePermissions;
}): SaveMonitoringDestinationRequest => {
  const forms = getDestinationClassForms(
    monitoringClass,
    values.authMode,
    getFormMode(original)
  );
  const parameters = toParameters(values, forms, original);
  const tls = toTls(values);
  const source = getCredentialsSource(values, original, secretPermissions);
  const keptSecretName = getKeptCredentialsSecretName(
    values.authMode,
    original
  );
  const credentialsSecret = toCredentialsSecretWrite(
    values,
    forms,
    source,
    keptSecretName
  );
  const secretName =
    source === CredentialsSource.Existing
      ? values.existingSecretName
      : (credentialsSecret?.name ?? keptSecretName);

  return {
    namespace: values.namespace,
    isEdit: !!original,
    credentialsSecret,
    destination: {
      ...(original && { apiVersion: original.apiVersion, kind: original.kind }),
      metadata: original ? { ...original.metadata } : { name: values.name },
      spec: {
        classRef: { name: values.className },
        authentication: {
          mode: values.authMode,
          ...(source &&
            secretName && { credentialsSecretRef: { name: secretName } }),
        },
        ...(tls && { tls }),
        ...(Object.keys(parameters).length > 0 && { parameters }),
      },
    },
  };
};

export const getSecretErrorMessage = (
  error: SecretAlreadyExistsError | SecretChangedMeanwhileError
) => {
  if (error instanceof SecretChangedMeanwhileError) {
    return Messages.secretChangedMeanwhile(error.secretName);
  }
  return error.reusable
    ? Messages.secretExists(error.secretName)
    : Messages.secretTaken(error.secretName);
};
