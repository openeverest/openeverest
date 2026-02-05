// everest
// Copyright (C) 2023 Percona LLC
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

import { Box, FormGroup, Stack, Tooltip } from '@mui/material';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { useCallback, useEffect, useMemo } from 'react';
import { lt, valid } from 'semver';
import { DbType } from '@percona/types';
import {
  ActionableLabeledContent,
  AutoCompleteInput,
  SwitchInput,
  TextInput,
} from '@percona/ui-lib';
import { dbTypeToDbEngine } from '@percona/utils';
import { useFormContext } from 'react-hook-form';
import { DbEngineToolStatus } from 'shared-types/dbEngines.types';
import { StepProps } from '../../../database-form.types.ts';
import { DbWizardFormFields } from 'consts.ts';
import { useDatabasePageMode } from '../../../useDatabasePageMode.ts';
import { StepHeader } from '../step-header/step-header.tsx';
import { Messages } from './first-step.messages.ts';
import { filterAvailableDbVersionsForDbEngineEdition } from 'components/cluster-form/db-version/utils.ts';
import { useNamespacePermissionsForResource } from 'hooks/rbac';
import { DbVersion } from 'components/cluster-form/db-version';
import { useDBEnginesForDbEngineTypes } from 'hooks/index.ts';
import { useDatabasePageDefaultValues } from 'pages/database-form/useDatabaseFormDefaultValues.ts';
import { getDbWizardDefaultValues } from 'pages/database-form/database-form.utils';
import { WizardMode } from 'shared-types/wizard.types.ts';

export const FirstStep = ({ loadingDefaultsForEdition }: StepProps) => {
  const mode = useDatabasePageMode();

  const {
    defaultValues: { [DbWizardFormFields.dbVersion]: defaultDbVersion },
  } = useDatabasePageDefaultValues(mode);
  const { watch, setValue, getFieldState, resetField, getValues } =
    useFormContext();

  const dbType: DbType = watch(DbWizardFormFields.dbType);
  const dbVersion: DbType = watch(DbWizardFormFields.dbVersion);
  const dbNamespace = watch(DbWizardFormFields.k8sNamespace);

  const [dbEnginesFoDbEngineTypes, dbEnginesFoDbEngineTypesFetching] =
    useDBEnginesForDbEngineTypes(dbTypeToDbEngine(dbType));

  const dbEnginesDataWithNamespaces = useMemo(() => {
    return !dbEnginesFoDbEngineTypesFetching
      ? dbEnginesFoDbEngineTypes.map((item) => item?.dbEngines).flat(1)
      : [];
  }, [dbEnginesFoDbEngineTypesFetching, dbEnginesFoDbEngineTypes]);

  const dbEngineData = useMemo(() => {
    const dbEnginesArray = dbEnginesDataWithNamespaces
      .filter((item) => item.namespace === dbNamespace)
      .map((item) => item.dbEngine);
    const dbEngine = dbEnginesArray ? dbEnginesArray[0] : undefined;
    if (mode !== WizardMode.New && dbEngine) {
      const validVersions = filterAvailableDbVersionsForDbEngineEdition(
        dbEngine,
        defaultDbVersion,
        mode
      );
      return {
        ...dbEngine,
        availableVersions: {
          ...dbEngine.availableVersions,
          engine: validVersions,
        },
      };
    }
    return dbEngine;
  }, [dbEnginesDataWithNamespaces, dbNamespace, mode, defaultDbVersion]);

  const notSupportedMongoOperatorVersionForSharding =
    dbType === DbType.Mongo &&
    !!valid(dbEngineData?.operatorVersion) &&
    lt(dbEngineData?.operatorVersion || '', '1.17.0');

  const disableSharding =
    mode !== WizardMode.New || notSupportedMongoOperatorVersionForSharding;

  const { canCreate, isLoading } =
    useNamespacePermissionsForResource('database-clusters');

  const filteredNamespaces = canCreate.filter((namespace) =>
    dbEnginesDataWithNamespaces?.find(
      (dbEngine) => dbEngine.namespace === namespace
    )
  );

  // setting the dbnamespace default value
  useEffect(() => {
    const { isTouched: k8sNamespaceTouched } = getFieldState(
      DbWizardFormFields.k8sNamespace
    );
    if (
      !k8sNamespaceTouched &&
      mode === WizardMode.New &&
      filteredNamespaces.length > 0 &&
      !isLoading
    ) {
      setValue(DbWizardFormFields.k8sNamespace, filteredNamespaces[0], {
        shouldValidate: true,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, isLoading, filteredNamespaces.length]);

  // setting the dvVersion default value
  useEffect(() => {
    const { isDirty: dbVersionDirty } = getFieldState(
      DbWizardFormFields.dbVersion
    );

    // Safety check
    if (
      dbVersionDirty ||
      !dbEngineData ||
      !dbEngineData.availableVersions.engine.length
    ) {
      return;
    }
    if (
      (mode === WizardMode.Restore && !dbVersion) ||
      mode === WizardMode.New
    ) {
      const recommendedVersion = dbEngineData.availableVersions.engine.find(
        (version) => version.status === DbEngineToolStatus.RECOMMENDED
      );

      setValue(
        DbWizardFormFields.dbVersion,
        recommendedVersion
          ? recommendedVersion.version
          : dbEngineData.availableVersions.engine[0].version,
        { shouldValidate: true }
      );
    }
  }, [dbVersion, dbEngineData, getFieldState, mode, setValue]);

  const onNamespaceChange = () => {
    const defaults = getDbWizardDefaultValues(dbType);
    setValue(
      DbWizardFormFields.monitoringInstance,
      defaults.monitoringInstance
    );
    setValue(DbWizardFormFields.monitoring, defaults.monitoring);
    setValue(
      DbWizardFormFields.monitoringInstance,
      defaults.monitoringInstance
    );
    setValue(DbWizardFormFields.schedules, []);
    setValue(DbWizardFormFields.pitrEnabled, false);
  };

  const onShardingToggleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const enabled = e.target.checked;

      if (!enabled) {
        resetField(DbWizardFormFields.shardNr, {
          keepError: false,
        });
        resetField(DbWizardFormFields.shardConfigServers, {
          keepError: false,
        });
      } else {
        setValue(DbWizardFormFields.splitHorizonDNSEnabled, false);
      }
    },
    [getFieldState, getValues, resetField, setValue]
  );

  return (
    <>
      <StepHeader
        pageTitle={Messages.pageTitle}
        pageDescription={Messages.pageDescription}
      />
      <FormGroup sx={{ mt: 3 }}>
        <AutoCompleteInput
          labelProps={{
            sx: { mt: 1 },
          }}
          name={DbWizardFormFields.k8sNamespace}
          label={Messages.labels.k8sNamespace}
          loading={isLoading}
          options={filteredNamespaces}
          disabled={mode === WizardMode.Restore || loadingDefaultsForEdition}
          onChange={onNamespaceChange}
          autoCompleteProps={{
            disableClearable: true,
            isOptionEqualToValue: (option, value) => option === value,
          }}
        />

        <TextInput
          name={DbWizardFormFields.dbName}
          label={Messages.labels.dbName}
          textFieldProps={{
            placeholder: Messages.placeholders.dbName,
            disabled: loadingDefaultsForEdition,
          }}
        />
        <DbVersion
          selectInputProps={{
            selectFieldProps: { disabled: mode === WizardMode.Restore },
          }}
          availableVersions={dbEngineData?.availableVersions.engine}
          loading={dbEnginesFoDbEngineTypesFetching || isLoading}
        />
        {dbType === DbType.Mongo && (
          <Box sx={{ marginY: '30px' }}>
            <ActionableLabeledContent
              label="Shards"
              techPreview
              caption="MongoDB shards are partitions of data that distribute load and improve database scalability and performance."
            >
              <Stack spacing={1} direction="row" alignItems="center">
                <SwitchInput
                  label={Messages.labels.shardedCluster}
                  name={DbWizardFormFields.sharding}
                  switchFieldProps={{
                    disabled: disableSharding,
                    onChange: onShardingToggleChange,
                  }}
                />
                {notSupportedMongoOperatorVersionForSharding && (
                  <Tooltip
                    title={Messages.disableShardingTooltip}
                    arrow
                    placement="right"
                  >
                    <InfoOutlinedIcon color="primary" />
                  </Tooltip>
                )}
              </Stack>
            </ActionableLabeledContent>
          </Box>
        )}
      </FormGroup>
    </>
  );
};
