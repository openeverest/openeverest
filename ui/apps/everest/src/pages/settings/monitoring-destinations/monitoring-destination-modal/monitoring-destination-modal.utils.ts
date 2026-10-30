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
  fromOpenApiParameterFormValues,
  getOpenApiParameterFields,
  OpenApiParameterFields,
  toOpenApiParameterFormValues,
} from 'components/openapi-parameters-fields';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import {
  deepMerge,
  isPlainObject,
} from 'components/ui-generator/utils/object-path';
import {
  CredentialsSecretWrite,
  SaveMonitoringDestinationRequest,
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
  isSecretBackedAuthMode,
  postprocessSectionForm,
  toMonitoringSectionForm,
} from 'utils/monitoring-classes';
import {
  getCredentialsSecretName,
  getDefaultCredentialsSecretName,
  getRotatedCredentialsSecretName,
} from 'utils/monitoring-destinations';
import {
  CREDENTIALS_SECTION_KEY,
  DESTINATION_SECTION_KEY,
} from './monitoring-destination-modal.constants';
import {
  DestinationClassForms,
  MonitoringDestinationFormValues,
} from './monitoring-destination-modal.types';

const NO_PARAMETER_FIELDS: OpenApiParameterFields = { fields: [], skipped: [] };

export const getDestinationClassForms = (
  monitoringClass: MonitoringClass | undefined,
  authMode: string,
  formMode: FormMode
): DestinationClassForms => {
  if (!monitoringClass) {
    return { parameterFields: NO_PARAMETER_FIELDS };
  }
  const { destination } = getMonitoringClassUiSchema(monitoringClass);
  const credentials =
    authMode && isSecretBackedAuthMode(authMode)
      ? getCredentialsSection(monitoringClass, authMode)
      : undefined;

  return {
    destination:
      destination &&
      toMonitoringSectionForm(DESTINATION_SECTION_KEY, destination, formMode),
    parameterFields: destination
      ? NO_PARAMETER_FIELDS
      : getOpenApiParameterFields(
          monitoringClass.spec.destinationParametersSchema?.openAPIV3Schema
        ),
    // New credentials are always typed in full, also when editing.
    credentials:
      credentials &&
      toMonitoringSectionForm(CREDENTIALS_SECTION_KEY, credentials),
  };
};

export const getFormMode = (original: MonitoringDestination | undefined) =>
  original ? FormMode.Edit : FormMode.New;

// Whether the save writes a credentials Secret: always for a new
// Secret-backed destination, and on edit when the mode changes, the
// destination has no Secret yet or the user replaces the credentials. A
// replacement goes to a new Secret the destination then switches to.
export const getCredentialsSecretAction = (
  {
    authMode,
    replaceCredentials,
  }: Pick<MonitoringDestinationFormValues, 'authMode' | 'replaceCredentials'>,
  original: MonitoringDestination | undefined
): 'create' | 'replace' | undefined => {
  if (!authMode || !isSecretBackedAuthMode(authMode)) {
    return undefined;
  }
  const secretName = original && getCredentialsSecretName(original);
  if (!original || !secretName) {
    return 'create';
  }
  return replaceCredentials || authMode !== original.spec.authentication?.mode
    ? 'replace'
    : undefined;
};

// The Secret the destination keeps when the user doesn't replace the
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

export const getFormDefaults = ({
  original,
  namespace,
  classes,
}: {
  original?: MonitoringDestination;
  namespace: string;
  classes: MonitoringClass[];
}): MonitoringDestinationFormValues => {
  const empty: MonitoringDestinationFormValues = {
    name: '',
    namespace,
    className: '',
    authMode: '',
    replaceCredentials: false,
    credentialsSecretName: '',
    credentials: {},
    spec: {},
    fallbackParameters: {},
    insecureSkipVerify: false,
    caSecretName: '',
    caSecretKey: '',
  };
  if (!original) {
    return empty;
  }

  const className = original.spec.classRef.name;
  const authMode = original.spec.authentication?.mode ?? '';
  const forms = getDestinationClassForms(
    classes.find((c) => getMonitoringClassName(c) === className),
    authMode,
    FormMode.Edit
  );
  const parameters: unknown = original.spec.parameters;
  const ca = original.spec.tls?.ca?.secretKeyRef;

  return {
    ...empty,
    name: original.metadata?.name ?? '',
    namespace: original.metadata?.namespace ?? namespace,
    className,
    authMode,
    credentials: forms.credentials
      ? getSectionFormDefaults(forms.credentials)
      : {},
    spec: forms.destination
      ? deepMerge(getSectionFormDefaults(forms.destination), {
          parameters: isPlainObject(parameters) ? parameters : {},
        })
      : {},
    fallbackParameters: toOpenApiParameterFormValues(
      parameters,
      forms.parameterFields
    ),
    insecureSkipVerify: !!original.spec.tls?.insecureSkipVerify,
    caSecretName: ca?.name ?? '',
    caSecretKey: ca?.key ?? '',
  };
};

const toStringData = (values: Record<string, unknown>) =>
  Object.fromEntries(
    Object.entries(values).map(([key, value]) => [
      key,
      typeof value === 'string' ? value : JSON.stringify(value),
    ])
  );

const toParameters = (
  values: MonitoringDestinationFormValues,
  forms: DestinationClassForms,
  original: MonitoringDestination | undefined
): Record<string, unknown> => {
  if (forms.destination) {
    const { parameters } = postprocessSectionForm(
      values.spec,
      forms.destination
    );
    return isPlainObject(parameters) ? parameters : {};
  }
  // Parameters the generated form can't edit are kept as they are.
  const current: unknown = original?.spec.parameters;
  const kept = isPlainObject(current)
    ? Object.fromEntries(
        Object.entries(current).filter(([key]) =>
          forms.parameterFields.skipped.includes(key)
        )
      )
    : {};
  return {
    ...kept,
    ...fromOpenApiParameterFormValues(
      values.fallbackParameters,
      forms.parameterFields
    ),
  };
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
  original: MonitoringDestination | undefined,
  reuseSecretName: string | undefined
): CredentialsSecretWrite | undefined => {
  const action = getCredentialsSecretAction(values, original);
  if (!action) {
    return undefined;
  }
  if (reuseSecretName) {
    return { action: 'reuse', name: reuseSecretName };
  }
  const stringData = toStringData(
    forms.credentials
      ? postprocessSectionForm(values.credentials, forms.credentials)
      : {}
  );
  if (action === 'replace') {
    return {
      action,
      name: getRotatedCredentialsSecretName(values.name),
      stringData,
      previousName: original && getCredentialsSecretName(original),
    };
  }
  return {
    action,
    name:
      values.credentialsSecretName ||
      getDefaultCredentialsSecretName(values.name),
    stringData,
  };
};

// The requests a submit makes. Credentials go to the Secret only; the
// destination merely references it.
export const buildSaveRequest = ({
  values,
  monitoringClass,
  original,
  reuseSecretName,
}: {
  values: MonitoringDestinationFormValues;
  monitoringClass: MonitoringClass | undefined;
  original?: MonitoringDestination;
  reuseSecretName?: string;
}): SaveMonitoringDestinationRequest => {
  const forms = getDestinationClassForms(
    monitoringClass,
    values.authMode,
    getFormMode(original)
  );
  const parameters = toParameters(values, forms, original);
  const tls = toTls(values);
  const credentialsSecret = toCredentialsSecretWrite(
    values,
    forms,
    original,
    reuseSecretName
  );
  const secretName = isSecretBackedAuthMode(values.authMode)
    ? (credentialsSecret?.name ??
      (original && getCredentialsSecretName(original)))
    : undefined;

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
          ...(secretName && { credentialsSecretRef: { name: secretName } }),
        },
        ...(tls && { tls }),
        ...(Object.keys(parameters).length > 0 && { parameters }),
      },
    },
  };
};
