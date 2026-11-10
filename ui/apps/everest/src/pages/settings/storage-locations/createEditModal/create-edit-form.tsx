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

import { useEffect, useMemo, useState } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Button,
  Divider,
  MenuItem,
  Typography,
} from '@mui/material';
import {
  AutoCompleteInput,
  RadioGroup,
  SelectInput,
  TextInput,
} from '@percona/ui-lib';
import {
  BackupStorageSecretConflictError,
  backupStorageSecretName,
  CA_BUNDLE_DEFINITION,
  usesOwnCaSecret,
} from 'api/backupStorage';
import { useNamespaces } from 'hooks/api/namespaces/useNamespaces';
import { useSecrets } from 'hooks/api/secrets';
import { useClusterName } from 'hooks/api/useClusterName';
import {
  BackupStorageClass,
  BackupStorageCRD,
  BackupStorageFormValues,
  BackupStorageSecretKind,
  CaSource,
  CredentialsSource,
  TlsMode,
} from 'shared-types/backupStorages.types';
import { getBackupStorageClassDisplayName } from 'utils/backup-storages';
import { Messages } from '../storage-locations.messages';
import { StorageLocationsFields } from '../storage-locations.types';
import { ClassFields } from './class-fields';
import { CurrentCredentialsFields } from './current-credentials-fields/current-credentials-fields';
import {
  authModeUsesSecret,
  canKeepCredentials,
  findBackupStorageClass,
  getCredentialsFields,
  getStorageFields,
} from './create-edit-modal.utils';
import { useCredentialsReadOnlyReason } from './hooks/use-credentials-read-only-reason';

const SecretSelect = ({
  name,
  label,
  secretNames,
  loading,
}: {
  name: string;
  label: string;
  secretNames: string[];
  loading: boolean;
}) => {
  const value: string = useWatch({ name });
  const options =
    value && !secretNames.includes(value)
      ? [value, ...secretNames]
      : secretNames;

  return (
    <SelectInput name={name} label={label} loading={loading} isRequired>
      {options.map((secretName) => (
        <MenuItem key={secretName} value={secretName}>
          {secretName}
        </MenuItem>
      ))}
    </SelectInput>
  );
};

interface CreateEditFormProps {
  isEditMode: boolean;
  backupStorageClasses?: BackupStorageClass[];
  loadingBackupStorageClasses?: boolean;
  previous?: BackupStorageCRD;
  secretConflict?: BackupStorageSecretConflictError;
}

export const CreateEditStorageForm = ({
  isEditMode,
  backupStorageClasses = [],
  loadingBackupStorageClasses = false,
  previous,
  secretConflict,
}: CreateEditFormProps) => {
  const cluster = useClusterName();
  const { control, getValues, setValue } =
    useFormContext<BackupStorageFormValues>();
  const { data: namespaces = [], isFetching: isNamespacesFetching } =
    useNamespaces();
  const [
    storageName,
    namespace,
    backupStorageClassName,
    authMode,
    credentialsSource,
    tlsMode,
    caSource,
  ] = useWatch({
    control,
    name: [
      StorageLocationsFields.name,
      StorageLocationsFields.namespace,
      StorageLocationsFields.backupStorageClass,
      StorageLocationsFields.authMode,
      StorageLocationsFields.credentialsSource,
      StorageLocationsFields.tlsMode,
      StorageLocationsFields.caSource,
    ],
  });
  // Each picker lists the Secrets that can fill its reference: credentials of
  // the class and mode, and CA bundles.
  const {
    data: credentialsSecrets = [],
    isFetching: loadingCredentialsSecrets,
  } = useSecrets(
    cluster,
    namespace,
    { backupStorageClass: backupStorageClassName, definition: authMode },
    { enabled: !!backupStorageClassName && !!authMode }
  );
  const { data: caSecrets = [], isFetching: loadingCASecrets } = useSecrets(
    cluster,
    namespace,
    { definition: CA_BUNDLE_DEFINITION }
  );
  const credentialsSecretNames = useMemo(
    () => credentialsSecrets.flatMap((secret) => secret.metadata?.name ?? []),
    [credentialsSecrets]
  );
  const caSecretNames = useMemo(
    () => caSecrets.flatMap((secret) => secret.metadata?.name ?? []),
    [caSecrets]
  );
  const [handledConflict, setHandledConflict] =
    useState<BackupStorageSecretConflictError>();

  const backupStorageClass = findBackupStorageClass(
    backupStorageClasses,
    backupStorageClassName
  );
  const authModes = useMemo(
    () => backupStorageClass?.spec.authModes.map((mode) => mode.name) ?? [],
    [backupStorageClass]
  );
  const { location, storage } = useMemo(
    () => getStorageFields(backupStorageClass),
    [backupStorageClass]
  );
  const credentialsFields = useMemo(
    () => getCredentialsFields(backupStorageClass, authMode),
    [backupStorageClass, authMode]
  );
  const usesSecret = authModeUsesSecret(backupStorageClass, authMode);
  const keepsCredentials = canKeepCredentials(previous, authMode);
  const replacesCredentials =
    keepsCredentials && credentialsSource === CredentialsSource.Current;
  const entersCredentials =
    !keepsCredentials &&
    credentialsFields.length > 0 &&
    credentialsSource === CredentialsSource.New;
  const credentialsReadOnlyReason = useCredentialsReadOnlyReason(
    namespace,
    previous?.spec.authentication.credentialsSecretRef?.name ?? '',
    replacesCredentials
  );
  const credentialsOptions = keepsCredentials
    ? [
        {
          label: Messages.createEditModal.currentCredentials,
          value: CredentialsSource.Current,
        },
        {
          label: Messages.createEditModal.otherSavedCredentials,
          value: CredentialsSource.Existing,
        },
      ]
    : [
        {
          label: Messages.createEditModal.enterCredentials,
          value: CredentialsSource.New,
        },
        {
          label: Messages.createEditModal.savedCredentials,
          value: CredentialsSource.Existing,
        },
      ];
  const defaultCredentialsSecretName = backupStorageSecretName(
    storageName,
    BackupStorageSecretKind.Credentials
  );
  const replacesOwnCa = usesOwnCaSecret(previous);
  const conflict =
    secretConflict !== handledConflict ? secretConflict : undefined;

  useEffect(() => {
    if (
      !isEditMode &&
      backupStorageClasses.length === 1 &&
      !getValues(StorageLocationsFields.backupStorageClass)
    ) {
      setValue(
        StorageLocationsFields.backupStorageClass,
        backupStorageClasses[0].metadata?.name ?? '',
        { shouldValidate: true }
      );
    }
  }, [backupStorageClasses, isEditMode, getValues, setValue]);

  useEffect(() => {
    if (
      !backupStorageClass ||
      authModes.includes(getValues(StorageLocationsFields.authMode))
    ) {
      return;
    }
    setValue(
      StorageLocationsFields.authMode,
      authModes.length === 1 ? authModes[0] : '',
      { shouldValidate: true }
    );
  }, [backupStorageClass, authModes, getValues, setValue]);

  // Only the storage's current credentials can be replaced in place, so a
  // change of auth mode enters new ones, and changing it back keeps them.
  useEffect(() => {
    const source = getValues(StorageLocationsFields.credentialsSource);
    if (!keepsCredentials && source === CredentialsSource.Current) {
      setValue(StorageLocationsFields.credentialsSource, CredentialsSource.New);
    } else if (keepsCredentials && source === CredentialsSource.New) {
      setValue(
        StorageLocationsFields.credentialsSource,
        CredentialsSource.Current
      );
    }
  }, [keepsCredentials, getValues, setValue]);

  const pickConflictingSecret = () => {
    if (!conflict) {
      return;
    }
    if (conflict.kind === BackupStorageSecretKind.Credentials) {
      setValue(
        StorageLocationsFields.credentialsSource,
        CredentialsSource.Existing
      );
      setValue(
        StorageLocationsFields.credentialsSecretName,
        conflict.secretName,
        { shouldValidate: true }
      );
    } else {
      setValue(StorageLocationsFields.caSource, CaSource.Existing);
      setValue(StorageLocationsFields.caSecretName, conflict.secretName, {
        shouldValidate: true,
      });
    }
    setHandledConflict(conflict);
  };

  // A taken name can be used only if it holds credentials of the class and
  // mode, or certificates, as the pickers list them.
  const conflictReusable =
    !!conflict &&
    (conflict.kind === BackupStorageSecretKind.CA
      ? caSecretNames
      : credentialsSecretNames
    ).includes(conflict.secretName);

  const conflictAlert = (kind: BackupStorageSecretKind) =>
    conflict?.kind === kind && (
      <Alert
        severity="error"
        sx={{ mt: 2 }}
        data-testid="secret-conflict-alert"
        action={
          conflictReusable && (
            <Button
              color="inherit"
              size="small"
              onClick={pickConflictingSecret}
            >
              {Messages.createEditModal.useSecret}
            </Button>
          )
        }
      >
        {conflictReusable
          ? Messages.createEditModal.secretConflict(kind, conflict.secretName)
          : Messages.createEditModal.secretTaken(kind, conflict.secretName)}
      </Alert>
    );

  return (
    <>
      <TextInput
        textFieldProps={{
          placeholder: Messages.createEditModal.placeholders.name,
          disabled: isEditMode,
        }}
        name={StorageLocationsFields.name}
        label={Messages.name}
        isRequired
        labelProps={{ sx: { mt: 0 } }}
      />
      <AutoCompleteInput
        name={StorageLocationsFields.namespace}
        label={Messages.namespace}
        disabled={isEditMode}
        loading={isNamespacesFetching}
        options={namespaces}
        isRequired
        textFieldProps={{
          placeholder: Messages.createEditModal.placeholders.namespace,
        }}
      />
      <Divider sx={{ mt: 4, mb: 2 }} />
      {!loadingBackupStorageClasses && backupStorageClasses.length === 0 ? (
        <Alert severity="info">{Messages.createEditModal.noClasses}</Alert>
      ) : (
        <SelectInput
          name={StorageLocationsFields.backupStorageClass}
          label={Messages.backupStorageClass}
          helperText={backupStorageClass?.spec.description}
          loading={loadingBackupStorageClasses}
          selectFieldProps={{ disabled: isEditMode }}
          isRequired
        >
          {backupStorageClasses.map((item) => {
            const name = item.metadata?.name ?? '';
            return (
              <MenuItem key={name} value={name}>
                {getBackupStorageClassDisplayName(item, name)}
              </MenuItem>
            );
          })}
        </SelectInput>
      )}
      <ClassFields fields={location} root="spec" readOnly={isEditMode} />
      <ClassFields fields={storage} root="spec" />
      {backupStorageClass && (
        <>
          <Divider sx={{ mt: 4, mb: 2 }} />
          <Typography variant="sectionHeading">
            {Messages.createEditModal.authentication}
          </Typography>
          <SelectInput
            name={StorageLocationsFields.authMode}
            label={Messages.createEditModal.authMode}
            isRequired
          >
            {authModes.map((mode) => (
              <MenuItem key={mode} value={mode}>
                {mode}
              </MenuItem>
            ))}
          </SelectInput>
          {usesSecret && (
            <>
              {keepsCredentials || credentialsFields.length > 0 ? (
                <RadioGroup
                  name={StorageLocationsFields.credentialsSource}
                  radioGroupFieldProps={{ sx: { mt: 2 } }}
                  options={credentialsOptions}
                />
              ) : (
                <Typography variant="caption" sx={{ mt: 2 }}>
                  {Messages.createEditModal.noCredentialsForm}
                </Typography>
              )}
              {replacesCredentials && (
                <CurrentCredentialsFields
                  fields={credentialsFields}
                  readOnlyReason={credentialsReadOnlyReason}
                />
              )}
              {entersCredentials && (
                <ClassFields
                  fields={credentialsFields}
                  root={StorageLocationsFields.credentials}
                />
              )}
              {!replacesCredentials && !entersCredentials && (
                <SecretSelect
                  name={StorageLocationsFields.credentialsSecretName}
                  label={Messages.createEditModal.credentialsSelect}
                  secretNames={credentialsSecretNames}
                  loading={loadingCredentialsSecrets}
                />
              )}
              {conflictAlert(BackupStorageSecretKind.Credentials)}
            </>
          )}
        </>
      )}
      <Accordion
        defaultExpanded={tlsMode !== TlsMode.System}
        disableGutters
        sx={{ mt: 4 }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle1">
            {Messages.createEditModal.advanced}
          </Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ display: 'flex', flexDirection: 'column' }}>
          {entersCredentials && (
            <TextInput
              name={StorageLocationsFields.credentialsSaveAs}
              label={Messages.createEditModal.saveCredentialsAs}
              textFieldProps={{
                placeholder: defaultCredentialsSecretName,
                helperText: Messages.createEditModal.saveCredentialsAsHelper(
                  defaultCredentialsSecretName
                ),
                sx: { mb: 2 },
              }}
            />
          )}
          <RadioGroup
            name={StorageLocationsFields.tlsMode}
            label={Messages.createEditModal.tls}
            radioGroupFieldProps={{ row: false }}
            options={[
              {
                label: Messages.createEditModal.tlsSystem,
                value: TlsMode.System,
              },
              {
                label: Messages.createEditModal.tlsCustomCA,
                value: TlsMode.CustomCA,
              },
              {
                label: Messages.createEditModal.tlsSkipVerify,
                value: TlsMode.SkipVerify,
              },
            ]}
          />
          {tlsMode === TlsMode.CustomCA && (
            <>
              <RadioGroup
                name={StorageLocationsFields.caSource}
                radioGroupFieldProps={{ row: false }}
                options={[
                  {
                    label: Messages.createEditModal.caPem,
                    value: CaSource.New,
                  },
                  {
                    label: Messages.createEditModal.caSaved,
                    value: CaSource.Existing,
                  },
                  {
                    label: Messages.createEditModal.caExternal,
                    value: CaSource.External,
                  },
                ]}
              />
              {caSource === CaSource.New && (
                <TextInput
                  name={StorageLocationsFields.caBundle}
                  label={Messages.createEditModal.caBundle}
                  isRequired
                  textFieldProps={{
                    multiline: true,
                    minRows: 4,
                    placeholder: '-----BEGIN CERTIFICATE-----',
                    helperText: replacesOwnCa
                      ? Messages.createEditModal.caReplacesOwn
                      : undefined,
                  }}
                />
              )}
              {caSource === CaSource.Existing && (
                <SecretSelect
                  name={StorageLocationsFields.caSecretName}
                  label={Messages.createEditModal.caSelect}
                  secretNames={caSecretNames}
                  loading={loadingCASecrets}
                />
              )}
              {caSource === CaSource.External && (
                <>
                  <TextInput
                    name={StorageLocationsFields.caSecretName}
                    label={Messages.createEditModal.caExternalName}
                    isRequired
                  />
                  <TextInput
                    name={StorageLocationsFields.caSecretKey}
                    label={Messages.createEditModal.caKey}
                    isRequired
                  />
                </>
              )}
              {conflictAlert(BackupStorageSecretKind.CA)}
            </>
          )}
          {tlsMode === TlsMode.SkipVerify && (
            <Alert severity="warning" sx={{ mt: 2 }}>
              {Messages.createEditModal.tlsSkipVerifyWarning}
            </Alert>
          )}
        </AccordionDetails>
      </Accordion>
    </>
  );
};
