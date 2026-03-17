// everest
// Copyright (C) 2023 Percona LLC
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

import { useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useBlocker, useNavigate } from 'react-router-dom';
import { zodResolver } from '@hookform/resolvers/zod';
import { Stack } from '@mui/material';
import {
  FormProvider,
  SubmitHandler,
  useForm,
  useWatch,
} from 'react-hook-form';
import { useCreateInstance } from 'hooks/api/instances/useCreateInstance';
import { useActiveBreakpoint } from 'hooks/utils/useActiveBreakpoint';
import { DbWizardType } from './database-form-schema';
import DatabaseFormCancelDialog from './database-form-cancel-dialog/index';
import DatabaseFormBody from './database-form-body';
import DatabaseFormSideDrawer from './database-form-side-drawer';
import { useDBClustersForNamespaces, useNamespaces } from 'hooks';
import { WizardMode } from 'shared-types/wizard.types';
import { ZodType } from 'zod';
import { useDatabasePageDefaultValues } from './hooks/use-database-form-default-values';
import { useDatabasePageMode } from './hooks/use-database-page-mode';
import { DatabaseFormProvider } from './database-form-context';
import { useSchema } from './hooks/use-schema';
import { useUiGenerator } from 'components/ui-generator/hooks/ui-generator';
import { useDbValidationSchema } from './hooks/use-db-validation-schema';
import { ImportFields } from 'components/cluster-form/import/import.types';
import { DbWizardFormFields } from 'consts';
import { getDefaultValues } from 'components/ui-generator/utils/default-values';
import { formSubmitPostProcessing } from './utils/form-submit-post-processing';
import {
  BASE_STEP_ID,
  IMPORT_STEP_ID,
} from './database-form-body/steps/constants';
import { getSectionStepId } from 'components/ui-generator/utils/section-step-id';

// When the user switches topology, new topology fields are absent from form
// data, causing Zod to report errors at the parent-object level instead of the
// leaf field. This helper deep-merges topology defaults into current values so
// that every nested object is properly initialised before re-validation.
const mergeTopologyDefaults = (
  current: Record<string, unknown>,
  defaults: Record<string, unknown>
): Record<string, unknown> => {
  const result: Record<string, unknown> = { ...current };
  for (const key of Object.keys(defaults)) {
    if (result[key] === undefined || result[key] === null) {
      result[key] = defaults[key];
    } else if (
      typeof defaults[key] === 'object' &&
      defaults[key] !== null &&
      !Array.isArray(defaults[key]) &&
      typeof result[key] === 'object' &&
      result[key] !== null &&
      !Array.isArray(result[key])
    ) {
      result[key] = mergeTopologyDefaults(
        result[key] as Record<string, unknown>,
        defaults[key] as Record<string, unknown>
      );
    }
  }
  return result;
};

const flattenErrorPaths = (
  obj: Record<string, unknown>,
  prefix = ''
): string[] => {
  if (!obj || typeof obj !== 'object') return prefix ? [prefix] : [];
  // A leaf RHF error node always has `message` or `type`
  if (obj.message !== undefined || obj.type !== undefined)
    return prefix ? [prefix] : [];
  return Object.keys(obj).flatMap((key) =>
    flattenErrorPaths(
      obj[key] as Record<string, unknown>,
      prefix ? `${prefix}.${key}` : key
    )
  );
};

export const DatabasePage = () => {
  const latestDataRef = useRef<DbWizardType | null>(null);
  const [activeStepId, setActiveStepId] = useState<string>(BASE_STEP_ID);
  const [formSubmitted, setFormSubmitted] = useState(false);

  const { mutate: createInstance, isPending: isCreating } = useCreateInstance();
  const location = useLocation();
  const navigate = useNavigate();

  const { isDesktop } = useActiveBreakpoint();
  const mode = useDatabasePageMode();

  const { uiSchema, topologies, hasMultipleTopologies } = useSchema();
  const defaultTopology = topologies[0] || '';

  const { defaultValues } = useDatabasePageDefaultValues(
    mode,
    uiSchema,
    defaultTopology
  );

  // Loading state - true if defaults are not yet ready
  const loadingClusterValues = !defaultValues;

  //TODO change to providers logic?
  const { data: namespaces = [] } = useNamespaces({
    refetchInterval: 10 * 1000,
  });
  const dbClustersResults = useDBClustersForNamespaces(
    namespaces.map((ns) => ({
      namespace: ns,
    }))
  );
  const dbClustersNamesList = useMemo(
    () =>
      Object.values(dbClustersResults)
        .map((item) => item.queryResult.data)
        .flat()
        .map((db) => ({
          name: db?.metadata?.name!,
          namespace: db?.metadata.namespace!,
        })),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [JSON.stringify(dbClustersResults)]
  );

  const hasImportStep = location.state?.showImport;

  const validationSchemaRef = useRef<ZodType<DbWizardType> | null>(null);

  const methods = useForm<DbWizardType>({
    mode: 'onChange',
    resolver: (data, context, options) => {
      if (!validationSchemaRef.current) {
        return { values: data, errors: {} };
      }
      const customResolver = zodResolver(validationSchemaRef.current, undefined, {
        mode: 'sync',
      });
      return customResolver(data, context, options);
    },
    // @ts-ignore
    defaultValues,
  });

  const {
    reset,
    formState: { isDirty, errors },
    clearErrors,
    handleSubmit,
    trigger,
    control,
    getValues,
  } = methods;

  const selectedTopology = useWatch({
    control,
    name: DbWizardFormFields.topology,
  });

  // ── topology switch: seed new-topology field defaults ────────────────────────
  // When the user picks a different topology the new topology's nested objects
  // (e.g. spec.components.configServer) are absent from the current form values.
  // Zod would then report an invalid_type error at the *parent* path rather than
  // at the individual field, making the error icon appear on the wrong step.
  // Merging the new topology's defaults ensures every nested object is present,
  // so validation errors surface at the correct leaf paths.
  const prevTopologyTypeRef = useRef<string | undefined>(undefined);
  useEffect(() => {
    const topologyType = selectedTopology;
    if (!topologyType || !uiSchema) return;
    if (prevTopologyTypeRef.current === undefined) {
      prevTopologyTypeRef.current = topologyType;
      return;
    }
    if (topologyType === prevTopologyTypeRef.current) return;
    prevTopologyTypeRef.current = topologyType;

    const topologyDefaults = getDefaultValues(uiSchema, topologyType);
    const merged = mergeTopologyDefaults(
      getValues() as Record<string, unknown>,
      topologyDefaults
    );
    reset(merged as DbWizardType, { keepDirty: true, keepIsSubmitted: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedTopology, uiSchema]);

  // Stable reference – avoids re-running useUiGenerator when uiSchema is falsy
  // TODO recheck rerender
  const stableUiSchema = useMemo(() => uiSchema || {}, [uiSchema]);

  const {
    sections,
    sectionsOrder,
    zodSchema: { schema: generatedZodSchema },
    sectionFieldMap,
  } = useUiGenerator(stableUiSchema, selectedTopology);

  const sectionKeys = useMemo(
    () => sectionsOrder || Object.keys(sections),
    [sectionsOrder, sections]
  );

  const orderedStepIds = useMemo(
    () => [
      BASE_STEP_ID,
      ...(hasImportStep ? [IMPORT_STEP_ID] : []),
      ...sectionKeys.map((key) => getSectionStepId(key)),
    ],
    [hasImportStep, sectionKeys]
  );

  const stepIdToIndex = useMemo(
    () =>
      Object.fromEntries(orderedStepIds.map((id, idx) => [id, idx])) as Record<
        string,
        number
      >,
    [orderedStepIds]
  );

  const activeStepIndex = stepIdToIndex[activeStepId] ?? 0;

  //TODO probably it will be cleaner to use steps.length or to
  // use Callback func (but in first scenario may be a problem of
  // last step cashing) and problem with submit/cancel
  const totalSteps = useMemo(() => orderedStepIds.length, [orderedStepIds]);

  const validationSchema = useDbValidationSchema(
    dbClustersNamesList,
    hasImportStep,
    generatedZodSchema
  ) as unknown as ZodType<DbWizardType>;

  useEffect(() => {
    validationSchemaRef.current = validationSchema;
    // Revalidate when schema changes
    trigger();
  }, [validationSchema, trigger]);

  useEffect(() => {
    if (generatedZodSchema && !loadingClusterValues && defaultValues) {
      trigger();
    }
  }, [generatedZodSchema, defaultValues, loadingClusterValues, trigger]);

  //TODO refactor and move to separate hook
  const fieldToStepMap = useMemo(() => {
    const map: Record<string, string> = {
      [DbWizardFormFields.dbName]: BASE_STEP_ID,
      [DbWizardFormFields.provider]: BASE_STEP_ID,
      [DbWizardFormFields.k8sNamespace]: BASE_STEP_ID,
      [DbWizardFormFields.topology]: BASE_STEP_ID,
    };

    if (hasImportStep) {
      (Object.values(ImportFields) as string[]).forEach((f) => {
        map[f] = IMPORT_STEP_ID;
      });
    }

    Object.entries(sectionFieldMap).forEach(([fieldPath, sectionKey]) => {
      map[fieldPath] = getSectionStepId(sectionKey);
    });

    return map;
  }, [hasImportStep, sectionFieldMap]);

  const stepsWithErrors = useMemo(() => {
    const errorPaths = flattenErrorPaths(errors as Record<string, unknown>);
    const stepsSet = new Set<string>();

    errorPaths.forEach((path) => {
      if (typeof path !== 'string' || path.length === 0) {
        if (import.meta.env.DEV) {
          console.error('[DatabasePage][stepsWithErrors] Invalid error path', {
            path,
            pathType: typeof path,
            errorPaths,
          });
        }
        return;
      }

      // Try the full path first, then progressively shorter prefixes.
      // After the buildSectionFieldMap fix, exact and intermediate paths are
      // registered, so the first match is almost always the full path.
      const parts = path.split('.');
      for (let i = parts.length; i > 0; i--) {
        const prefix = parts.slice(0, i).join('.');
        if (prefix in fieldToStepMap) {
          stepsSet.add(fieldToStepMap[prefix]);
          break;
        }
      }
    });

    return Array.from(stepsSet).filter((stepId) => stepId in stepIdToIndex);
  }, [errors, fieldToStepMap, stepIdToIndex]);

  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      isDirty &&
      !formSubmitted &&
      currentLocation.pathname !== nextLocation.pathname
  );

  const onSubmit: SubmitHandler<DbWizardType> = (data) => {
    const postProcessedData = formSubmitPostProcessing(
      {},
      data as Record<string, unknown>,
      {
        schema: stableUiSchema,
        selectedTopology,
      }
    ) as DbWizardType;

    latestDataRef.current = postProcessedData;

    //TODO Restore mode === WizardMode.Restore
    if (mode === WizardMode.New) {
      const addInstance = () =>
        createInstance(
          {
            formValue: postProcessedData,
          },
          {
            onSuccess: () => {
              // TODO recheck with list of instances
              // // We clear the query for the namespace to make sure the new cluster is fetched
              // queryClient.removeQueries({
              //   queryKey: [DB_CLUSTERS_QUERY_KEY, instance.metadata.namespace],
              // });
              setFormSubmitted(true);
            },
          }
        );
      addInstance();
      //TODO import flow
      // const credentials = latestDataRef.current?.credentials;
      // if (hasImportStep && credentials && Object.keys(credentials).length > 0) {
      //   addDbClusterSecret(
      //     {
      //       dbClusterName: data.dbName,
      //       namespace: data.k8sNamespace || '',
      //       credentials: credentials as Record<string, string>,
      //     },
      //     {
      //       onSuccess: addCluster,
      //     }
      //   );
      // } else {
      //   addCluster();
      // }
    }
  };

  const handleNext = async () => {
    if (activeStepIndex < totalSteps - 1) {
      const newIndex = activeStepIndex + 1;
      setActiveStepId(orderedStepIds[newIndex]);
    }
  };

  const handleBack = () => {
    clearErrors();
    if (activeStepIndex > 0) {
      setActiveStepId(orderedStepIds[activeStepIndex - 1]);
    }
  };

  const handleSectionEdit = (stepId: string) => {
    clearErrors();
    setActiveStepId(stepId);
  };

  const handleCloseCancellationModal = () => {
    if (blocker.state === 'blocked') {
      blocker.reset();
    }
  };

  const proceedNavigation = () => {
    if (blocker.state === 'blocked') {
      blocker.proceed();
    }
  };

  useEffect(() => {
    trigger();
  }, [activeStepId, trigger]);

  useEffect(() => {
    if (!(activeStepId in stepIdToIndex)) {
      setActiveStepId(BASE_STEP_ID);
    }
  }, [activeStepId, stepIdToIndex]);

  useEffect(() => {
    // We disable the inputs on first step to make sure user doesn't change anything before all data is loaded
    // When users change the inputs, it means all data was loaded and we should't change the defaults anymore at this point
    // Because this effect relies on defaultValues, which comes from a hook that has dependencies that might be triggered somewhere else
    // E.g. If defaults depend on monitoringInstances query, step four will cause this to re-rerender, because that step calls that query again
    if (isDirty) {
      return;
    }

    if (mode === WizardMode.Restore) {
      reset(defaultValues);
    }
  }, [defaultValues, isDirty, reset, mode]);

  useEffect(() => {
    if (!location.state) {
      navigate('/');
    }
  }, []);

  useEffect(() => {
    if (formSubmitted) {
      navigate('/databases');
    }
  }, [formSubmitted, navigate]);

  if (!uiSchema) {
    //TODO check with noSchema in the api
    return;
  }

  //TODO move provider separately (clean code issue)
  return (
    <DatabaseFormProvider
      value={{
        uiSchema,
        topologies,
        hasMultipleTopologies,
        defaultTopology,
        sections,
        sectionsOrder,
        providerObject: location.state?.selectedDbProvider,
      }}
    >
      <Stack direction={isDesktop ? 'row' : 'column'}>
        <FormProvider {...methods}>
          <DatabaseFormBody
            activeStep={activeStepIndex}
            isSubmitting={isCreating}
            hasErrors={stepsWithErrors.length > 0}
            disableNext={
              hasImportStep &&
              activeStepId === IMPORT_STEP_ID &&
              stepsWithErrors.includes(IMPORT_STEP_ID)
            }
            onSubmit={handleSubmit(onSubmit)}
            onCancel={() => navigate('/databases')}
            handleNextStep={handleNext}
            handlePreviousStep={handleBack}
          />
          <DatabaseFormSideDrawer
            disabled={loadingClusterValues}
            activeStepId={activeStepId}
            handleSectionEdit={handleSectionEdit}
            stepsWithErrors={stepsWithErrors}
          />
        </FormProvider>
      </Stack>
      <DatabaseFormCancelDialog
        open={blocker.state === 'blocked'}
        onClose={handleCloseCancellationModal}
        onConfirm={proceedNavigation}
      />
    </DatabaseFormProvider>
  );
};
