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
  getAuthenticationFormValues,
  getAuthenticationRequest,
  SecretNamespacePermissions,
  toCredentialsForm,
} from 'components/class-authentication';
import {
  getOpenApiParameterFields,
  OpenApiParameterFields,
  toOpenApiParameterFormValues,
  toOpenApiParametersUpdate,
} from 'components/openapi-parameters-fields';
import {
  getTlsConfigFormValues,
  toTlsConfig,
} from 'components/tls-config-fields';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { isPlainObject } from 'components/ui-generator/utils/object-path';
import {
  getSectionFormDefaults,
  getSectionFormUpdate,
  getSectionFormValues,
  toSectionForm,
} from 'components/ui-generator/utils/section-form';
import { SaveMonitoringDestinationRequest } from 'hooks/api/monitoring-destinations';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { toParametersContract } from 'utils/class-contract';
import {
  getMonitoringClassName,
  getMonitoringClassUiSchema,
  toMonitoringClassContract,
} from 'utils/monitoring-classes';
import { getDefaultCredentialsSecretName } from 'utils/monitoring-destinations';
import { DESTINATION_SECTION_KEY } from './monitoring-destination-modal.constants';
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

  return {
    destination:
      destination &&
      toSectionForm(DESTINATION_SECTION_KEY, destination, {
        formMode,
        contract: toParametersContract(
          monitoringClass.spec.destinationParametersSchema
        ),
      }),
    parameterFields: destination
      ? NO_PARAMETER_FIELDS
      : getOpenApiParameterFields(
          monitoringClass.spec.destinationParametersSchema?.openAPIV3Schema
        ),
    credentials: toCredentialsForm(
      toMonitoringClassContract(monitoringClass),
      authMode
    ),
  };
};

export const getFormMode = (original: MonitoringDestination | undefined) =>
  original ? FormMode.Edit : FormMode.New;

// The class-dependent fields of a new destination of the class.
export const getNewDestinationClassDefaults = (
  monitoringClass: MonitoringClass | undefined
): Pick<
  MonitoringDestinationFormValues,
  'authMode' | 'spec' | 'fallbackParameters' | 'credentials'
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
    fallbackParameters: toOpenApiParameterFormValues(
      undefined,
      forms.parameterFields
    ),
    credentials: forms.credentials
      ? getSectionFormDefaults(forms.credentials)
      : {},
  };
};

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
    ...getAuthenticationFormValues(undefined),
    spec: {},
    fallbackParameters: {},
    ...getTlsConfigFormValues(undefined),
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
  const authentication = original.spec.authentication;
  const forms = getDestinationClassForms(
    classes.find((c) => getMonitoringClassName(c) === originalClassName),
    authentication?.mode ?? '',
    FormMode.Edit
  );

  return {
    ...empty,
    name: original.metadata?.name ?? '',
    namespace: original.metadata?.namespace ?? namespace,
    className: originalClassName,
    ...getAuthenticationFormValues(authentication, forms.credentials),
    spec: forms.destination
      ? getSectionFormValues(forms.destination, original.spec)
      : {},
    fallbackParameters: toOpenApiParameterFormValues(
      original.spec.parameters,
      forms.parameterFields
    ),
    ...getTlsConfigFormValues(original.spec.tls),
  };
};

const toParameters = (
  values: MonitoringDestinationFormValues,
  forms: DestinationClassForms,
  original: MonitoringDestination | undefined
): Record<string, unknown> => {
  if (forms.destination) {
    // An edit keeps the parameters the form doesn't own as they were saved.
    const { parameters } = getSectionFormUpdate(
      forms.destination,
      values.spec,
      original?.spec ?? {}
    );
    return isPlainObject(parameters) ? parameters : {};
  }
  return toOpenApiParametersUpdate(
    values.fallbackParameters,
    forms.parameterFields,
    original?.spec.parameters
  );
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
  const tls = toTlsConfig(values);
  const { authentication, credentialsSecret } = getAuthenticationRequest({
    values,
    contract: monitoringClass && toMonitoringClassContract(monitoringClass),
    credentialsForm: forms.credentials,
    saved: original?.spec.authentication,
    permissions: secretPermissions,
    defaultSecretName: getDefaultCredentialsSecretName(values.name),
  });

  return {
    namespace: values.namespace,
    isEdit: !!original,
    credentialsSecret,
    destination: {
      ...(original && { apiVersion: original.apiVersion, kind: original.kind }),
      metadata: original ? { ...original.metadata } : { name: values.name },
      spec: {
        classRef: { name: values.className },
        authentication,
        ...(tls && { tls }),
        ...(Object.keys(parameters).length > 0 && { parameters }),
      },
    },
  };
};
