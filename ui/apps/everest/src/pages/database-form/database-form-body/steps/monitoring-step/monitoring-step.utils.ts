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

import { z } from 'zod';
import { removeEmptyFieldValues } from 'components/ui-generator/utils/postprocess/postprocess-schema';
import {
  deleteByPathAndEmptyParents,
  getByPath,
  isPlainObject,
  setByPath,
} from 'components/ui-generator/utils/object-path';
import {
  InstanceMonitoringDestination,
  InstanceMonitoringDestinationPayload,
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { getDestinationName } from 'utils/monitoring-destinations';
import {
  getMonitoringClassName,
  getMonitoringClassUiSchema,
  getSectionFormDefaults,
  getSectionFormZodSchema,
  MonitoringSectionForm,
  postprocessSectionForm,
  toMonitoringSectionForm,
} from 'utils/monitoring-classes';
import {
  MAX_MONITORING_DESTINATIONS,
  MONITORING_DESTINATIONS_FIELD,
  MONITORING_ENTRY_NAME_MAX_LENGTH,
  MONITORING_INSTANCE_SECTION_KEY,
} from './monitoring-step.constants';
import {
  DestinationDisabledReason,
  MonitoringClassDestinations,
} from './monitoring-step.types';

const isInstanceMonitoringDestination = (
  value: unknown
): value is InstanceMonitoringDestination => {
  if (!isPlainObject(value) || typeof value.name !== 'string') {
    return false;
  }
  const ref = value.destinationRef;
  return isPlainObject(ref) && typeof ref.name === 'string';
};

export const parseMonitoringDestinations = (
  value: unknown
): InstanceMonitoringDestination[] =>
  Array.isArray(value) ? value.filter(isInstanceMonitoringDestination) : [];

// Keeps the order of the classes and lists every class, including those
// without a destination in the namespace yet.
export const groupDestinationsByClass = (
  classes: MonitoringClass[],
  destinations: MonitoringDestination[]
): MonitoringClassDestinations[] =>
  classes.map((monitoringClass) => ({
    monitoringClass,
    destinations: destinations.filter(
      (destination) =>
        destination.spec.classRef.name ===
        getMonitoringClassName(monitoringClass)
    ),
  }));

const toDnsLabel = (name: string) =>
  name
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, '-')
    .slice(0, MONITORING_ENTRY_NAME_MAX_LENGTH)
    .replace(/^-+|-+$/g, '');

// Entry names default to the destination name, made a DNS label and unique
// among the Instance's entries.
export const toMonitoringEntryName = (
  destinationName: string,
  takenNames: string[]
): string => {
  const base = toDnsLabel(destinationName) || 'destination';
  if (!takenNames.includes(base)) {
    return base;
  }
  for (let index = 2; ; index++) {
    const suffix = `-${index}`;
    const candidate = `${base.slice(0, MONITORING_ENTRY_NAME_MAX_LENGTH - suffix.length).replace(/-+$/, '')}${suffix}`;
    if (!takenNames.includes(candidate)) {
      return candidate;
    }
  }
};

export const isDestinationSelected = (
  selected: Pick<InstanceMonitoringDestination, 'destinationRef'>[],
  destinationName: string
) => selected.some((entry) => entry.destinationRef.name === destinationName);

// A new entry starts from the defaults of the class's instance form, if any.
export const toggleMonitoringDestination = (
  selected: InstanceMonitoringDestinationPayload[],
  destinationName: string,
  defaultParameters?: Record<string, unknown>
): InstanceMonitoringDestinationPayload[] => {
  if (isDestinationSelected(selected, destinationName)) {
    return selected.filter(
      (entry) => entry.destinationRef.name !== destinationName
    );
  }
  return [
    ...selected,
    {
      name: toMonitoringEntryName(
        destinationName,
        selected.map((entry) => entry.name)
      ),
      destinationRef: { name: destinationName },
      ...(defaultParameters && { parameters: defaultParameters }),
    },
  ];
};

// The integration a class is rendered through for the provider, when the
// provider renders it natively (e.g. PMM through spec.pmm of the engine CR).
const getProviderManagedIntegration = (
  monitoringClass: MonitoringClass,
  provider: string
): string | undefined => {
  const compatibility = monitoringClass.status?.providers?.find(
    (entry) => entry.name === provider
  );
  return compatibility?.mode === 'ProviderManaged'
    ? monitoringClass.spec.providerManaged?.integration
    : undefined;
};

// Why a destination cannot be added to the selection, if it cannot. Core
// keeps one ProviderManaged entry per integration, because the engine
// operator runs one agent of each kind.
export const getDestinationDisabledReason = ({
  destination,
  selected,
  destinations,
  classes,
  provider,
}: {
  destination: MonitoringDestination;
  selected: InstanceMonitoringDestination[];
  destinations: MonitoringDestination[];
  classes: MonitoringClass[];
  provider: string;
}): DestinationDisabledReason | undefined => {
  const name = getDestinationName(destination);
  if (isDestinationSelected(selected, name)) {
    return undefined;
  }
  if (selected.length >= MAX_MONITORING_DESTINATIONS) {
    return DestinationDisabledReason.LimitReached;
  }

  const integrationOf = (classRefName: string) => {
    const monitoringClass = classes.find(
      (c) => getMonitoringClassName(c) === classRefName
    );
    return monitoringClass
      ? getProviderManagedIntegration(monitoringClass, provider)
      : undefined;
  };

  const integration = integrationOf(destination.spec.classRef.name);
  if (!integration) {
    return undefined;
  }
  const integrationTaken = selected.some((entry) => {
    const selectedDestination = destinations.find(
      (d) => getDestinationName(d) === entry.destinationRef.name
    );
    return (
      !!selectedDestination &&
      integrationOf(selectedDestination.spec.classRef.name) === integration
    );
  });
  return integrationTaken
    ? DestinationDisabledReason.IntegrationTaken
    : undefined;
};

// Drops entries whose destination is not offered, e.g. after the namespace
// changed on an earlier step.
export const pruneMonitoringDestinations = (
  selected: InstanceMonitoringDestination[],
  offeredDestinationNames: string[]
): InstanceMonitoringDestination[] =>
  selected.filter((entry) =>
    offeredDestinationNames.includes(entry.destinationRef.name)
  );

// The instance form of each destination whose class defines one.
export const buildInstanceFormsByDestination = (
  classes: MonitoringClass[],
  destinations: MonitoringDestination[]
): Map<string, MonitoringSectionForm> => {
  const formsByClass = new Map(
    classes.flatMap((monitoringClass): [string, MonitoringSectionForm][] => {
      const { instance } = getMonitoringClassUiSchema(monitoringClass);
      return instance
        ? [
            [
              getMonitoringClassName(monitoringClass),
              toMonitoringSectionForm(
                MONITORING_INSTANCE_SECTION_KEY,
                instance
              ),
            ],
          ]
        : [];
    })
  );
  return new Map(
    destinations.flatMap((destination): [string, MonitoringSectionForm][] => {
      const form = formsByClass.get(destination.spec.classRef.name);
      return form ? [[getDestinationName(destination), form]] : [];
    })
  );
};

export const getDefaultEntryParameters = (
  form: MonitoringSectionForm | undefined
): Record<string, unknown> | undefined => {
  const parameters = form ? getSectionFormDefaults(form).parameters : undefined;
  return isPlainObject(parameters) ? parameters : undefined;
};

// Each entry's parameters are validated against its class's instance form;
// issues point at the entry's own fields in the wizard.
export const buildMonitoringParametersSchema = (
  formsByDestination: Map<string, MonitoringSectionForm>
): z.ZodTypeAny => {
  const schemasByDestination = new Map(
    [...formsByDestination].map(([destinationName, form]) => [
      destinationName,
      getSectionFormZodSchema(form),
    ])
  );
  const entriesPath = MONITORING_DESTINATIONS_FIELD.split('.');

  return z.unknown().superRefine((data, ctx) => {
    if (!isPlainObject(data)) {
      return;
    }
    parseMonitoringDestinations(
      getByPath(data, MONITORING_DESTINATIONS_FIELD)
    ).forEach((entry, index) => {
      const result = schemasByDestination
        .get(entry.destinationRef.name)
        ?.safeParse(entry);
      result?.error?.issues.forEach((issue) =>
        ctx.addIssue({ ...issue, path: [...entriesPath, index, ...issue.path] })
      );
    });
  });
};

// The entries as the Instance spec takes them: parameters cleaned like any
// ui-generator form and left out when nothing was filled.
export const finalizeMonitoringEntries = (
  entries: InstanceMonitoringDestination[],
  formsByDestination: Map<string, MonitoringSectionForm>
): InstanceMonitoringDestinationPayload[] =>
  entries.map(({ name, destinationRef, parameters }) => {
    const form = formsByDestination.get(destinationRef.name);
    const values = form
      ? postprocessSectionForm({ parameters }, form)
      : removeEmptyFieldValues({ parameters });
    const cleaned = values.parameters;
    return {
      name,
      destinationRef: { name: destinationRef.name },
      ...(isPlainObject(cleaned) &&
        Object.keys(cleaned).length > 0 && { parameters: cleaned }),
    };
  });

// Writes the finalized entries into the create request, or drops
// spec.monitoring when no destination is selected.
export const writeMonitoringEntries = (
  formData: Record<string, unknown>,
  formsByDestination: Map<string, MonitoringSectionForm>
) => {
  const entries = parseMonitoringDestinations(
    getByPath(formData, MONITORING_DESTINATIONS_FIELD)
  );
  if (entries.length === 0) {
    deleteByPathAndEmptyParents(formData, MONITORING_DESTINATIONS_FIELD);
    return;
  }
  setByPath(
    formData,
    MONITORING_DESTINATIONS_FIELD,
    finalizeMonitoringEntries(entries, formsByDestination)
  );
};
