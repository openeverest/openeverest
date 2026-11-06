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

import { useEffect, useMemo, useRef } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import { toOpenApiParameterFormValues } from 'components/openapi-parameters-fields';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { getSectionFormDefaults } from 'utils/monitoring-classes';
import { DestinationFormFields } from '../monitoring-destination-modal.constants';
import {
  DestinationClassForms,
  MonitoringDestinationFormValues,
} from '../monitoring-destination-modal.types';
import {
  getDestinationClassForms,
  getFormMode,
} from '../monitoring-destination-modal.utils';

// The forms of the picked class and auth mode. Picking another class starts
// its forms from their defaults (and preselects a class's only auth mode);
// picking another mode clears the credentials typed for the previous one.
export const useDestinationClassForms = (
  classes: MonitoringClass[],
  original: MonitoringDestination | undefined
): {
  monitoringClass: MonitoringClass | undefined;
  forms: DestinationClassForms;
} => {
  const { control, setValue } =
    useFormContext<MonitoringDestinationFormValues>();
  const [className, authMode] = useWatch({
    control,
    name: [DestinationFormFields.className, DestinationFormFields.authMode],
  });
  const monitoringClass = classes.find((c) => c.metadata?.name === className);
  const formMode = getFormMode(original);
  const forms = useMemo(
    () => getDestinationClassForms(monitoringClass, authMode, formMode),
    [monitoringClass, authMode, formMode]
  );

  const previousClassName = useRef(className);
  useEffect(() => {
    if (previousClassName.current === className) {
      return;
    }
    previousClassName.current = className;
    const authModes = monitoringClass?.spec.authModes ?? [];
    const classForms = getDestinationClassForms(monitoringClass, '', formMode);
    setValue(
      DestinationFormFields.authMode,
      authModes.length === 1 ? authModes[0].name : '',
      { shouldValidate: true }
    );
    setValue(
      DestinationFormFields.spec,
      classForms.destination
        ? getSectionFormDefaults(classForms.destination)
        : {}
    );
    setValue(
      DestinationFormFields.fallbackParameters,
      toOpenApiParameterFormValues(undefined, classForms.parameterFields)
    );
  }, [className, monitoringClass, formMode, setValue]);

  const previousAuthMode = useRef(authMode);
  useEffect(() => {
    if (previousAuthMode.current === authMode) {
      return;
    }
    previousAuthMode.current = authMode;
    setValue(
      DestinationFormFields.credentials,
      forms.credentials ? getSectionFormDefaults(forms.credentials) : {}
    );
  }, [authMode, forms.credentials, setValue]);

  return { monitoringClass, forms };
};
