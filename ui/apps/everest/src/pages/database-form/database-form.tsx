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
import { useDbValidationSchema } from './hooks/use-db-validation-schema';
import { ImportFields } from 'components/cluster-form/import/import.types';
import { DbWizardFormFields } from 'consts';
import { getDefaultValues } from 'components/ui-generator/utils/default-values';
import {
  BASE_STEP_ID,
  IMPORT_STEP_ID,
} from './database-form-body/steps/constants';
import {
  useFormEngine,
  useStepNavigation,
  useErrorRouting,
  StepDefinition,
} from 'components/ui-generator/form-engine';
import { BaseInfoStep } from './database-form-body/steps/base-step/base-step';
import { ImportStep } from './database-form-body/steps-old/import/import-step';
import { mergeTopologyDefaults } from 'components/ui-generator/utils/default-values/merge-topology-defaults';

export const DatabasePage = () => {
  const latestDataRef = useRef<DbWizardType | null>(null);
  const [formSubmitted, setFormSubmitted] = useState(false);

  const { mutate: createInstance, isPending: isCreating } = useCreateInstance();
  const location = useLocation();
  const navigate = useNavigate();

  const { isDesktop } = useActiveBreakpoint();
  const mode = useDatabasePageMode();

  // ── Schema & topology
  const { uiSchema, topologies, hasMultipleTopologies } = useSchema();
  const defaultTopology = topologies[0] || '';
  const hasImportStep = !!location.state?.showImport;
  const providerObject = location.state?.selectedDbProvider;

  // ── Page-level defaults (merges schema defaults + wizard-specific ones)
  const { defaultValues } = useDatabasePageDefaultValues(
    mode,
    uiSchema,
    defaultTopology
  );
  const loadingClusterValues = !defaultValues;

  // ── Data queries ─────────────────────────────────────────────────────────
  const { data: namespaces = [] } = useNamespaces({
    refetchInterval: 10 * 1000,
  });
  const dbClustersResults = useDBClustersForNamespaces(
    namespaces.map((ns) => ({ namespace: ns }))
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

  // ── React Hook Form ──────────────────────────────────────────────────────
  const validationSchemaRef = useRef<ZodType<DbWizardType> | null>(null);

  const methods = useForm<DbWizardType>({
    mode: 'onChange',
    resolver: (data, context, options) => {
      if (!validationSchemaRef.current) {
        return { values: data, errors: {} };
      }
      return zodResolver(validationSchemaRef.current, undefined, {
        mode: 'sync',
      })(data, context, options);
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

  // Static step definitions
  const staticSteps = useMemo((): StepDefinition[] => {
    const steps: StepDefinition[] = [
      {
        id: BASE_STEP_ID,
        label: 'Basic Info',
        component: BaseInfoStep,
        fields: [
          DbWizardFormFields.dbName,
          DbWizardFormFields.provider,
          DbWizardFormFields.k8sNamespace,
          DbWizardFormFields.topology,
        ],
      },
    ];
    if (hasImportStep) {
      steps.push({
        id: IMPORT_STEP_ID,
        label: 'Import',
        component: ImportStep,
        fields: Object.values(ImportFields) as string[],
      });
    }
    return steps;
  }, [hasImportStep]);

  const engine = useFormEngine({
    uiSchema,
    selectedTopology,
    staticSteps,
    providerObject,
  });

  // Navigation
  const nav = useStepNavigation(engine.steps, BASE_STEP_ID);

  const handleNext = () => nav.next();
  const handleBack = () => {
    clearErrors();
    nav.back();
  };
  const handleSectionEdit = (stepId: string) => {
    clearErrors();
    nav.goTo(stepId);
  };

  // Error step routing
  const validStepIds = useMemo(
    () => new Set(engine.steps.map((s) => s.id)),
    [engine.steps]
  );
  const stepsWithErrors = useErrorRouting(
    errors,
    engine.fieldToStepMap,
    validStepIds
  );

  // Validation
  const validationSchema = useDbValidationSchema(
    dbClustersNamesList,
    hasImportStep,
    engine.zodSchema
  ) as unknown as ZodType<DbWizardType>;

  useEffect(() => {
    validationSchemaRef.current = validationSchema;
    trigger();
  }, [validationSchema, trigger]);

  useEffect(() => {
    if (engine.zodSchema && !loadingClusterValues && defaultValues) {
      trigger();
    }
  }, [engine.zodSchema, defaultValues, loadingClusterValues, trigger]);

  // Topology switch
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

  // Revalidate on step change
  useEffect(() => {
    trigger();
  }, [nav.activeStepId, trigger]);

  // Restore mode defaults
  useEffect(() => {
    if (isDirty) return;
    if (mode === WizardMode.Restore) {
      reset(defaultValues);
    }
  }, [defaultValues, isDirty, reset, mode]);

  // Route guards
  useEffect(() => {
    if (!location.state) navigate('/');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (formSubmitted) navigate('/databases');
  }, [formSubmitted, navigate]);

  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      isDirty &&
      !formSubmitted &&
      currentLocation.pathname !== nextLocation.pathname
  );

  const onSubmit: SubmitHandler<DbWizardType> = (data) => {
    const postProcessedData = engine.postprocess(
      data as Record<string, unknown>
    ) as DbWizardType;

    latestDataRef.current = postProcessedData;

    if (mode === WizardMode.New) {
      createInstance(
        { formValue: postProcessedData },
        {
          onSuccess: () => {
            setFormSubmitted(true);
          },
        }
      );
    }
  };

  // ── Render ───────────────────────────────────────────────────────────────
  if (!uiSchema) return null;

  return (
    <DatabaseFormProvider
      value={{
        uiSchema,
        topologies,
        hasMultipleTopologies,
        defaultTopology,
        sections: engine.sections,
        sectionsOrder: engine.sectionsOrder,
        providerObject,
      }}
    >
      <Stack direction={isDesktop ? 'row' : 'column'}>
        <FormProvider {...methods}>
          <DatabaseFormBody
            steps={engine.steps}
            activeStep={nav.activeStepIndex}
            isSubmitting={isCreating}
            hasErrors={stepsWithErrors.length > 0}
            disableNext={
              hasImportStep &&
              nav.activeStepId === IMPORT_STEP_ID &&
              stepsWithErrors.includes(IMPORT_STEP_ID)
            }
            onSubmit={handleSubmit(onSubmit)}
            onCancel={() => navigate('/databases')}
            handleNextStep={handleNext}
            handlePreviousStep={handleBack}
          />
          <DatabaseFormSideDrawer
            disabled={loadingClusterValues}
            activeStepId={nav.activeStepId}
            handleSectionEdit={handleSectionEdit}
            stepsWithErrors={stepsWithErrors}
          />
        </FormProvider>
      </Stack>
      <DatabaseFormCancelDialog
        open={blocker.state === 'blocked'}
        onClose={() => blocker.state === 'blocked' && blocker.reset()}
        onConfirm={() => blocker.state === 'blocked' && blocker.proceed()}
      />
    </DatabaseFormProvider>
  );
};
