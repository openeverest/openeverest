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

import { FormGroup, MenuItem } from '@mui/material';
import { useEffect } from 'react';
import { AutoCompleteInput, SelectInput, TextInput } from '@percona/ui-lib';
import { useFormContext } from 'react-hook-form';
import { StepProps } from '../../../database-form.types.js';
import { DbWizardFormFields } from 'consts.ts';
import { useDatabasePageMode } from '../../../hooks/use-database-page-mode.js';
import { StepHeader } from '../../steps-old/step-header/step-header.js';
import { Messages } from '../../steps-old/first/first-step.messages.js';
import { useNamespacePermissionsForResource } from 'hooks/rbac';
import { useNamespaces } from 'hooks/index.ts';
import { WizardMode } from 'shared-types/wizard.types.ts';
import { useDatabaseFormContext } from 'pages/database-form/database-form-context';

export const BaseInfoStep = ({ loadingDefaultsForEdition }: StepProps) => {
  const mode = useDatabasePageMode();
  // TODO probably we should show only list of available namespaces for provider, needs to be checked
  const { data: namespaces = [] } = useNamespaces({
    refetchInterval: 10 * 1000,
  });
  const { topologies, hasMultipleTopologies } = useDatabaseFormContext();
  const { watch, setValue, getFieldState } = useFormContext();

  // const dbType: DbType = watch(DbWizardFormFields.dbType);
  // const dbNamespace = watch(DbWizardFormFields.k8sNamespace);
  const currentTopology = watch(DbWizardFormFields.topology);

  // const [dbEnginesFoDbEngineTypes, dbEnginesFoDbEngineTypesFetching] =
  //   useDBEnginesForDbEngineTypes(dbTypeToDbEngine(dbType));

  // const dbEnginesDataWithNamespaces = useMemo(() => {
  //   return !dbEnginesFoDbEngineTypesFetching
  //     ? dbEnginesFoDbEngineTypes.map((item) => item?.dbEngines).flat(1)
  //     : [];
  // }, [dbEnginesFoDbEngineTypesFetching, dbEnginesFoDbEngineTypes]);

  // const dbEngineData = useMemo(() => {
  //   const dbEnginesArray = dbEnginesDataWithNamespaces
  //     .filter((item) => item.namespace === dbNamespace)
  //     .map((item) => item.dbEngine);
  //   const dbEngine = dbEnginesArray ? dbEnginesArray[0] : undefined;
  //   if (mode !== WizardMode.New && dbEngine) {
  //     const validVersions = filterAvailableDbVersionsForDbEngineEdition(
  //       dbEngine,
  //       defaultDbVersion,
  //       mode
  //     );
  //     return {
  //       ...dbEngine,
  //       availableVersions: {
  //         ...dbEngine.availableVersions,
  //         engine: validVersions,
  //       },
  //     };
  //   }
  //   return dbEngine;
  // }, [dbEnginesDataWithNamespaces, dbNamespace, mode, defaultDbVersion]);

  //TODO know will it be includet to the api or not
  // const notSupportedMongoOperatorVersionForSharding =
  //   dbType === DbType.Mongo &&
  //   !!valid(dbEngineData?.operatorVersion) &&
  //   lt(dbEngineData?.operatorVersion || '', '1.17.0');

  const { isLoading } = useNamespacePermissionsForResource('database-clusters');

  // const filteredNamespaces = canCreate.filter((namespace) =>
  //   dbEnginesDataWithNamespaces?.find(
  //     (dbEngine) => dbEngine.namespace === namespace
  //   )
  // );

  // setting the dbnamespace default value
  useEffect(() => {
    const { isTouched: k8sNamespaceTouched } = getFieldState(
      DbWizardFormFields.k8sNamespace
    );
    if (
      !k8sNamespaceTouched &&
      mode === WizardMode.New &&
      namespaces.length > 0 &&
      !isLoading
    ) {
      setValue(DbWizardFormFields.k8sNamespace, namespaces[0], {
        shouldValidate: true,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, isLoading, namespaces.length]);

  // setting the topology default value
  useEffect(() => {
    const { isTouched: topologyTouched } = getFieldState(
      DbWizardFormFields.topology
    );
    if (
      !topologyTouched &&
      mode === WizardMode.New &&
      topologies.length > 0 &&
      !currentTopology
    ) {
      setValue(DbWizardFormFields.topology, topologies[0], {
        shouldValidate: true,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, topologies.length, currentTopology]);

  // TODO remember the logic of recommended versions, ask team about it and implement
  // const onNamespaceChange = () => {
  // TODO discuss with the team this case, should we keep the same
  // logic or not? Reason of this logic was in the problem of
  // availability of storage, monitoring of namespaces (RBAC issue)
  // const defaults = getDbWizardDefaultValues(dbType);
  // setValue(DbWizardFormFields.schedules, []);
  // setValue(DbWizardFormFields.pitrEnabled, false);
  // };

  //TODO check and add test for dynimic part of the form, that if some field was added and then removed because of poor topology, so it shouldn't be a part of the api body

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
          options={namespaces}
          disabled={mode === WizardMode.Restore || loadingDefaultsForEdition}
          // onChange={onNamespaceChange}
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
        {hasMultipleTopologies && (
          <SelectInput
            name={DbWizardFormFields.topology}
            label="Database Topology"
            selectFieldProps={{
              disabled: loadingDefaultsForEdition,
            }}
          >
            {topologies.map((topology) => (
              <MenuItem key={topology} value={topology}>
                {topology}
              </MenuItem>
            ))}
          </SelectInput>
        )}
      </FormGroup>
    </>
  );
};
export default BaseInfoStep;
