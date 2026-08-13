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

import { useEffect, useMemo, useRef, useState, useCallback } from 'react';
import { useLocation, useBlocker, useNavigate } from 'react-router-dom';
import { zodResolver } from '@hookform/resolvers/zod';
import { Stack } from '@mui/material';
import { useQueryClient } from '@tanstack/react-query';
import {
  FormProvider,
  SubmitHandler,
  useForm,
  useWatch,
} from 'react-hook-form';
import { useCreateDbInstance } from 'hooks/api/db-instances/useCreateDbInstance';
import { DB_INSTANCES_QUERY_KEY } from 'hooks/api/db-instances/useDbInstanceList';
import { buildRestoreDataSource } from 'hooks/api/restores/restore-data-source';
import { useActiveBreakpoint } from 'hooks/utils/useActiveBreakpoint';
import { DbWizardType } from './database-form-schema';
import DatabaseFormCancelDialog from './database-form-cancel-dialog/index';
import DatabaseFormBody from './database-form-body';
import DatabaseFormSideDrawer from './database-form-side-drawer';
import { useInstancesForNamespaces, useNamespaces } from 'hooks';

import { FormMode } from 'components/ui-generator/ui-generator.types';
import { ZodType } from 'zod';
import { useDatabasePageDefaultValues } from './hooks/use-database-form-default-values';
import { useDatabasePageMode } from './hooks/use-database-page-mode';
import { useRestoreNavigationState } from './hooks/use-restore-navigation-state';
import { DatabaseFormProvider } from './database-form-context';
import { useSchema } from './hooks/use-schema';
import { useDbValidationSchema } from './hooks/use-db-validation-schema';
import { ImportFields } from 'components/cluster-form/import/import.types';
import { DbWizardFormFields } from 'consts';
import { getDefaultValues } from 'components/ui-generator/utils/default-values';
import {
  BASE_STEP_ID,
  IMPORT_STEP_ID,
  BACKUP_STEP_ID,
} from './database-form-body/steps/constants';
import {
  useFormEngine,
  useStepNavigation,
  useErrorRouting,
  StepDefinition,
} from 'components/ui-generator/form-engine';
import { DataSourcePrefetcher } from 'components/ui-generator/api-providers';
import { BaseInfoStep } from './database-form-body/steps/base-step/base-step';
import { ImportStep } from './database-form-body/steps-old/import/import-step';
import {
  BackupStep,
  BACKUP_SCHEDULES_FIELD,
  buildBackupSpecFromWizard,
  ensureStorageRegistered,
  WizardBackupSpec,
  WizardPitrMap,
} from './database-form-body/steps/backup-step';
import { FlattenedSchedule } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog-context.types';
import { useBackupClassesList } from 'hooks/api/backup-classes/useBackupClasses';
import { useClusterName } from 'hooks/api/useClusterName';
import { mergeTopologyDefaults } from 'components/ui-generator/utils/default-values/merge-topology-defaults';
import { PluginFormSections } from './plugin-form-sections';
import { useSubmitPluginInstanceConfig } from 'hooks/api/plugins/useSubmitPluginInstanceConfig';

type UnknownRecord = Record<string, unknown>;

type ParsedBackupFormData = {
  schedules: FlattenedSchedule[];
  classRefName?: string;
  pitr?: WizardPitrMap;
};

const isRecord = (value: unknown): value is UnknownRecord =>
  typeof value === 'object' && value !== null;

const isFlattenedSchedule = (value: unknown): value is FlattenedSchedule => {
  if (!isRecord(value)) {
    return false;
  }

  const name = Reflect.get(value, 'name');
  const cron = Reflect.get(value, 'cron');
  const enabled = Reflect.get(value, 'enabled');
  const storageName = Reflect.get(value, 'storageName');
  const retentionCopies = Reflect.get(value, 'retentionCopies');
  const parameters = Reflect.get(value, 'parameters');

  const hasCoreFields =
    typeof name === 'string' &&
    typeof cron === 'string' &&
    typeof enabled === 'boolean' &&
    typeof storageName === 'string';

  const hasOptionalFields =
    (retentionCopies === undefined || typeof retentionCopies === 'number') &&
    (parameters === undefined || isRecord(parameters));

  return hasCoreFields && hasOptionalFields;
};

const parseWizardPitrMap = (value: unknown): WizardPitrMap | undefined => {
  if (!isRecord(value)) {
    return undefined;
  }

  const pitrMap: WizardPitrMap = {};

  for (const [storageName, config] of Object.entries(value)) {
    if (!isRecord(config)) {
      continue;
    }

    const enabled = Reflect.get(config, 'enabled');
    const parameters = Reflect.get(config, 'parameters');

    if (typeof enabled !== 'boolean') {
      continue;
    }

    pitrMap[storageName] = {
      enabled,
      ...(isRecord(parameters) ? { parameters } : {}),
    };
  }

  return pitrMap;
};

const parseBackupFormData = (
  value: unknown
): ParsedBackupFormData | undefined => {
  if (!isRecord(value)) {
    return undefined;
  }

  const schedulesRaw = Reflect.get(value, 'schedules');
  const schedules = Array.isArray(schedulesRaw)
    ? schedulesRaw.filter(isFlattenedSchedule)
    : undefined;

  if (!schedules) {
    return undefined;
  }

  const classRef = Reflect.get(value, 'classRef');
  const classRefName = isRecord(classRef)
    ? Reflect.get(classRef, 'name')
    : undefined;

  return {
    schedules,
    classRefName: typeof classRefName === 'string' ? classRefName : undefined,
    pitr: parseWizardPitrMap(Reflect.get(value, 'pitr')),
  };
};

export const DatabasePage = () => {
  const latestDataRef = useRef<UnknownRecord | null>(null);
  const [formSubmitted, setFormSubmitted] = useState(false);

  const { mutate: createInstance, isPending: isCreating } =
    useCreateDbInstance();
  const { mutate: submitPluginConfig } = useSubmitPluginInstanceConfig();
  const pluginConfigsRef = useRef<Record<string, Record<string, unknown>>>({});

  const handlePluginConfigChange = useCallback(
    (pluginName: string, config: Record<string, unknown>) => {
      pluginConfigsRef.current[pluginName] = config;
    },
    []
  );
  const location = useLocation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { isDesktop } = useActiveBreakpoint();
  const mode = useDatabasePageMode();
  const { restoreDataSource } = useRestoreNavigationState();

  // ── Schema & topology
  const { uiSchema, topologies, hasMultipleTopologies, resolvedProvider } =
    useSchema();
  const defaultTopology = topologies[0] || '';
  const hasImportStep = !!location.state?.showImport;
  const providerObject = location.state?.selectedDbProvider ?? resolvedProvider;

  // ── Backup classes (determines if backup step is shown)
  const clusterName = useClusterName();
  const { data: backupClasses = [] } = useBackupClassesList(clusterName);
  const hasBackupStep = backupClasses.length > 0;

  // ── Page-level defaults (merges schema defaults + wizard-specific ones)
  const { defaultValues, dbClusterRequestStatus } =
    useDatabasePageDefaultValues(
      mode,
      uiSchema,
      defaultTopology,
      hasBackupStep
    );
  const loadingClusterValues = !defaultValues;

  // ── Data queries ─────────────────────────────────────────────────────────
  const { data: namespaces = [] } = useNamespaces({
    refetchInterval: 10 * 1000,
  });
  const dbInstancesResults = useInstancesForNamespaces(
    namespaces.map((ns) => ({ namespace: ns }))
  );
  const dbInstancesList = useMemo(
    () =>
      Object.values(dbInstancesResults)
        .map((item) => item.queryResult.data)
        .flat()
        .filter((instance): instance is NonNullable<typeof instance> =>
          Boolean(instance)
        )
        .flatMap((instance) => {
          const name = instance.metadata?.name;
          const namespace = instance.metadata?.namespace;

          return name && namespace ? [{ name, namespace }] : [];
        }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [JSON.stringify(dbInstancesResults)]
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

  const selectedNamespace = useWatch({
    control,
    name: DbWizardFormFields.k8sNamespace,
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
    if (hasBackupStep) {
      steps.push({
        id: BACKUP_STEP_ID,
        label: 'Backups',
        component: BackupStep,
        fields: [BACKUP_SCHEDULES_FIELD],
      });
    }
    return steps;
  }, [hasImportStep, hasBackupStep]);

  const engine = useFormEngine({
    uiSchema,
    selectedTopology,
    staticSteps,
    providerObject,
    namespace: selectedNamespace || namespaces[0],
    formMode: mode,
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
    dbInstancesList,
    hasImportStep,
    engine.zodSchema
  ) as unknown as ZodType<DbWizardType>;

  useEffect(() => {
    validationSchemaRef.current = validationSchema;
  }, [validationSchema]);

  // Topology switch — must run BEFORE the revalidation effect so that
  // form values are reset before trigger() validates them.
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

  // Revalidate after validation schema changes or defaults finish loading.
  // Declared after the topology switch effect so that reset() runs first.
  useEffect(() => {
    if (validationSchemaRef.current && !loadingClusterValues) {
      trigger();
    }
  }, [validationSchema, loadingClusterValues, trigger]);

  // Revalidate on step change
  useEffect(() => {
    trigger();
  }, [nav.activeStepId, trigger]);

  // Restore mode defaults — force reset when source instance data arrives.
  // We track whether the "real" (instance-backed) defaults have been applied,
  // and skip the isDirty guard the first time they arrive.
  const restoreDefaultsApplied = useRef(false);
  useEffect(() => {
    if (mode !== FormMode.Restore) return;
    if (dbClusterRequestStatus !== 'success') return;
    if (restoreDefaultsApplied.current && isDirty) return;
    restoreDefaultsApplied.current = true;
    reset(defaultValues);
  }, [defaultValues, isDirty, reset, mode, dbClusterRequestStatus]);

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
      !isCreating &&
      currentLocation.pathname !== nextLocation.pathname
  );

  const onSubmit: SubmitHandler<DbWizardType> = (data) => {
    const postProcessedData = engine.postprocess(data);

    // Transform flat backup schedules into nested storages structure
    const formData = postProcessedData;
    const backupFormData = parseBackupFormData(Reflect.get(formData, 'backup'));
    let backupSpecForSubmit: WizardBackupSpec | undefined;
    if (backupFormData) {
      const backupSpec = buildBackupSpecFromWizard(
        backupFormData.schedules,
        backupFormData.classRefName,
        backupFormData.pitr
      );
      backupSpecForSubmit = backupSpec;
      if (backupSpec) {
        Reflect.set(formData, 'backup', backupSpec);
      } else {
        Reflect.deleteProperty(formData, 'backup');
      }
    }

    latestDataRef.current = postProcessedData;

    if (mode === FormMode.New) {
      createInstance(
        { formValue: postProcessedData },
        {
          onSuccess: () => {
            // Submit plugin configs for each plugin that provided config.
            const instanceName = data.dbName || '';
            const ns = data.k8sNamespace || '';
            for (const [pluginName, config] of Object.entries(
              pluginConfigsRef.current
            )) {
              if (Object.keys(config).length > 0) {
                submitPluginConfig({
                  pluginName,
                  instanceName,
                  namespace: ns,
                  config,
                });
              }
            }
            // mark the instances list stale so the databases page
            // (which mounts right after we navigate) renders the
            // freshly created row instead of the stale, pre-create cache.
            queryClient.invalidateQueries({
              queryKey: [DB_INSTANCES_QUERY_KEY],
            });
            setFormSubmitted(true);
          },
        }
      );
    } else if (mode === FormMode.Restore) {
      if (!restoreDataSource) return;
      // Seed the new instance from the source in a single atomic create: the
      // operator runs the restore from spec.dataSource (it creates the seeding
      // Restore CR itself), so there is no second request to orchestrate here.
      Reflect.set(
        formData,
        'dataSource',
        buildRestoreDataSource(restoreDataSource)
      );
      // The seeding storage (where the operator reads the source) must be
      // registered on the new instance regardless of the user's schedule
      // choices, or the restore stalls waiting for it.
      Reflect.set(
        formData,
        'backup',
        ensureStorageRegistered(
          backupSpecForSubmit,
          restoreDataSource.storageName,
          backupFormData?.classRefName
        )
      );
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
  if (!uiSchema || (mode === FormMode.Restore && topologies.length === 0)) {
    return null;
  }

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
        hasBackupStep,
      }}
    >
      <Stack direction={isDesktop ? 'row' : 'column'}>
        <FormProvider {...methods}>
          <DataSourcePrefetcher
            sections={engine.sections}
            namespace={selectedNamespace || namespaces[0]}
          />
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
          {nav.activeStepIndex === 0 && (
            <PluginFormSections
              formValues={methods.getValues() as Record<string, unknown>}
              namespace={selectedNamespace || namespaces[0] || ''}
              engineType={providerObject?.name}
              isCreate={true}
              onPluginConfigChange={handlePluginConfigChange}
            />
          )}
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
