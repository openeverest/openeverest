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

import { SelectInput } from '@percona/ui-lib';
import { Box, MenuItem } from '@mui/material';
import { FormCardWithCheck, FormCardWithDialog } from 'components/form-card';
import { Messages } from './messages';
import { ImportFields } from './import.types';
import { S3DetailsSection } from './sections/s3-details/s3-details-section';
import { SectionKeys } from './sections/constants';
import { FileDirectorySection } from './sections/file-directory/file-directory-section';
import { DbCredentialsSection } from './sections/db-credentials/db-credentials-section';
import { useEffect, useMemo } from 'react';
import { DbWizardFormFields } from 'consts';
import { useFormContext } from 'react-hook-form';
import { dbTypeToDbEngine } from '@percona/utils';
import { useDataImporters } from 'hooks/api/data-importers/useDataImporters';
import { useDbEngine, useDbEngines } from 'hooks';

export const ImportForm = () => {
  const { getValues, watch, setValue } = useFormContext();

  const dbType = dbTypeToDbEngine(getValues(DbWizardFormFields.dbType));
  const namespace = getValues(DbWizardFormFields.k8sNamespace);
  const selectedImporter = watch(DbWizardFormFields.dataImporter);

  const { data: dataImporters } = useDataImporters(dbType);

  const showCreds = useMemo(() => {
    const selectedImporterInfo = (dataImporters?.items || []).find(
      (i) => i.metadata.name === selectedImporter
    );
    return (
      selectedImporterInfo?.spec.databaseClusterConstraints.requiredFields || []
    ).includes('.spec.engine.userSecretsName');
  }, [dataImporters, selectedImporter]);

  const { data: engines = [] } = useDbEngines(namespace);
  const selectedEngineName = engines.find(
    (engine) => engine.type === dbType
  )?.name;
  const { data: engine } = useDbEngine(namespace, selectedEngineName || '');
  const secretKeys = engine?.spec?.secretKeys;

  const defaultDataImporter =
    dataImporters?.items?.length === 1
      ? dataImporters.items[0].metadata.name
      : '';

  useEffect(() => {
    if (defaultDataImporter) {
      setValue(ImportFields.dataImporter, defaultDataImporter);
    } else {
      setValue(ImportFields.dataImporter, '');
    }
  }, [defaultDataImporter, setValue]);

  useEffect(() => {
    setValue('showCreds', showCreds);
  }, [showCreds, setValue]);

  return (
    <Box data-testid="import-form">
      <>
        <FormCardWithCheck
          title={Messages.dataImporter.label}
          controlComponent={
            <SelectInput
              controllerProps={{
                name: ImportFields.dataImporter,
              }}
              name={ImportFields.dataImporter}
              label={Messages.dataImporter.placeholder}
              formControlProps={{ sx: { marginTop: 0 } }}
              selectFieldProps={{
                sx: { minWidth: '240px' },
                MenuProps: {
                  anchorOrigin: {
                    vertical: 'bottom',
                    horizontal: 'left',
                  },
                },
              }}
            >
              {dataImporters?.items.map((dataImporter) => (
                <MenuItem
                  value={dataImporter.metadata.name}
                  key={dataImporter.metadata.name}
                >
                  {dataImporter.metadata.name}
                </MenuItem>
              ))}
            </SelectInput>
          }
        />

        <FormCardWithDialog
          title={Messages.s3Details.label}
          content={<S3DetailsSection />}
          sectionSavedKey={SectionKeys.s3Details}
        />

        <FormCardWithDialog
          title={Messages.fileDir.label}
          content={<FileDirectorySection />}
          sectionSavedKey={SectionKeys.fileDir}
        />
        {showCreds && (
          <FormCardWithDialog
            title={Messages.dbCreds.label}
            content={
              <DbCredentialsSection
                secretKeys={(secretKeys?.user || []).sort((a, b) =>
                  a.name.localeCompare(b.name)
                )}
              />
            }
            sectionSavedKey={SectionKeys.dbCreds}
          />
        )}
      </>
    </Box>
  );
};

export default ImportForm;
