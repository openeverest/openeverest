import { Alert, MenuItem, Typography } from '@mui/material';
import { DbType } from '@percona/types';
import {
  DateTimePickerInput,
  LoadableChildren,
  RadioGroup,
  SelectInput,
} from '@percona/ui-lib';
import { dbEngineToDbType } from '@percona/utils';
import ActionableAlert from 'components/actionable-alert';
import { FormDialog } from 'components/form-dialog';
import { FormDialogProps } from 'components/form-dialog/form-dialog.types';
import { DATE_FORMAT, PITR_DATE_FORMAT } from 'consts';
import { format } from 'date-fns';
import {
  BACKUPS_QUERY_KEY,
  useDbBackups,
  useDbClusterPitr,
} from 'hooks/api/backups/useBackups';
import {
  useDbClusterRestoreFromBackup,
  useDbClusterRestoreFromPointInTime,
} from 'hooks/api/restores/useDbClusterRestore';
import { FieldValues, useFormContext } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import {
  Backup,
  BackupStatus,
  DatabaseClusterPitr,
} from 'shared-types/backups.types';
import { DbCluster } from 'shared-types/dbCluster.types';
import {
  BackuptypeValues,
  RestoreDbFields,
  defaultValues,
  schema,
} from './restore-db-modal-schema';
import { Messages } from './restore-db-modal.messages';
import { useEffect, useMemo } from 'react';

const ModalContent = ({
  backupName,
  header,
  dbType,
  isLoading,
  pitrData,
  backups,
  backupStorageName,
}: {
  backupName?: string;
  isLoading: boolean;
  header: string;
  dbType: DbType;
  pitrData?: DatabaseClusterPitr;
  backups: Backup[];
  backupStorageName?: string;
}) => {
  const { watch, resetField, setValue, getValues } = useFormContext();
  const backupType: BackuptypeValues = watch(RestoreDbFields.backupType);

  useEffect(() => {
    if (!pitrData) {
      return;
    }
    if (!getValues(RestoreDbFields.pitrBackup)) {
      setValue(RestoreDbFields.pitrBackup, pitrData.latestDate);
    }
  }, [getValues, pitrData, setValue]);

  return (
    <LoadableChildren loading={isLoading}>
      <Typography variant="body1">{header}</Typography>
      <RadioGroup
        name={RestoreDbFields.backupType}
        radioGroupFieldProps={{
          sx: {
            ml: 1,
            display: 'flex',
            gap: 3,
            '& label': {
              display: 'flex',
              gap: '10px',
              alignItems: 'center',
              padding: 1,
              '& span': {
                padding: '0px !important',
              },
            },
          },
        }}
        options={[
          {
            label: Messages.fromBackup,
            value: BackuptypeValues.fromBackup,
            radioProps: {
              onClick: () => {
                resetField(RestoreDbFields.pitrBackup, {
                  keepError: false,
                  defaultValue: getValues(RestoreDbFields.pitrBackup),
                });
              },
            },
          },
          {
            label: Messages.fromPitr,
            value: BackuptypeValues.fromPitr,
            radioProps: {
              onClick: () => {
                resetField(RestoreDbFields.backupName, {
                  keepError: false,
                });
              },
            },
            disabled:
              !!backupName &&
              pitrData?.latestBackupName !== watch(RestoreDbFields.backupName),
          },
        ]}
      />
      {backupType === BackuptypeValues.fromBackup ? (
        <SelectInput
          label={Messages.selectBackup}
          name={RestoreDbFields.backupName}
          selectFieldProps={{
            labelId: 'restore-backup',
            label: Messages.selectBackup,
          }}
        >
          {backups
            .filter((value) => value.state === BackupStatus.OK)
            .sort((a, b) => {
              if (a.created && b.created) {
                return (
                  new Date(b.created).valueOf() - new Date(a.created).valueOf()
                );
              }
              return -1;
            })
            .map((value) => {
              const valueWithTime = `${
                value.name
              } - ${format(value.created!, DATE_FORMAT)}`;
              return (
                <MenuItem key={value.name} value={value.name}>
                  {valueWithTime}
                </MenuItem>
              );
            })}
        </SelectInput>
      ) : (
        <>
          {pitrData && dbType === DbType.Postresql && (
            <ActionableAlert
              sx={{ mt: 1.5 }}
              message={Messages.pitrLimitationAlert}
              buttonMessage={Messages.seeDocs}
              onClick={() =>
                window.open(
                  'https://openeverest.io/documentation/current/reference/known_limitations.html#postgresql-limitation-for-pitr',
                  '_blank',
                  'noopener'
                )
              }
              buttonProps={{
                sx: { whiteSpace: 'nowrap' },
              }}
            />
          )}
          {pitrData && (
            <Alert
              sx={{ mt: 1.5, mb: 1.5 }}
              severity={pitrData?.gaps ? 'error' : 'info'}
            >
              {pitrData?.gaps
                ? Messages.gapDisclaimer
                : Messages.pitrDisclaimer(
                    format(
                      pitrData?.earliestDate || new Date(),
                      PITR_DATE_FORMAT
                    ),
                    format(
                      pitrData?.latestDate || new Date(),
                      PITR_DATE_FORMAT
                    ),
                    backupStorageName || ''
                  )}
            </Alert>
          )}

          {!pitrData?.gaps && (
            <DateTimePickerInput
              ampm={false}
              views={['year', 'month', 'day', 'hours', 'minutes', 'seconds']}
              timeSteps={{ hours: 1, minutes: 1, seconds: 1 }}
              disableFuture
              disabled={!pitrData}
              minDate={new Date(pitrData?.earliestDate || new Date())}
              maxDate={new Date(pitrData?.latestDate || new Date())}
              format={PITR_DATE_FORMAT}
              name={RestoreDbFields.pitrBackup}
              label={pitrData ? 'Select point in time' : 'No options'}
              sx={{ mt: 1.5 }}
            />
          )}
        </>
      )}
    </LoadableChildren>
  );
};

const RestoreDbModal = <T extends FieldValues>({
  closeModal,
  isOpen,
  dbCluster,
  namespace,
  isNewClusterMode,
  backupName,
}: Pick<FormDialogProps<T>, 'closeModal' | 'isOpen'> & {
  dbCluster: DbCluster;
  namespace: string;
  isNewClusterMode: boolean;
  backupName?: string;
}) => {
  const navigate = useNavigate();
  // We use a memo to prevent changes in dbCluster from causing changes in PITR data
  // Even though gcTime is now set to 0 to prevent old data from populating the input, we avoid that situation
  const cluster = useMemo(() => dbCluster, []);
  // we use a different query key for the restore modal to avoid re-renders from "useDbBackups" running in the background
  const { data: backups = [], isLoading } = useDbBackups(
    cluster.metadata.name,
    namespace,
    {
      queryKey: [
        BACKUPS_QUERY_KEY,
        namespace,
        cluster.metadata.name,
        'restore-modal',
      ],
    }
  );
  const { data: pitrData } = useDbClusterPitr(
    cluster.metadata.name,
    namespace,
    {
      queryKey: [cluster.metadata.name, namespace, 'pitr', 'restore-modal'],
      gcTime: 0,
    }
  );

  const { mutate: restoreBackupFromBackup, isPending: restoringFromBackup } =
    useDbClusterRestoreFromBackup(cluster.metadata.name);
  const {
    mutate: restoreBackupFromPointInTime,
    isPending: restoringFromPointInTime,
  } = useDbClusterRestoreFromPointInTime(cluster.metadata.name);

  const pitrSchema = useMemo(
    () =>
      schema(!!pitrData?.gaps, pitrData?.earliestDate, pitrData?.latestDate),
    [pitrData]
  );

  return (
    <FormDialog
      size="XXXL"
      isOpen={isOpen}
      dataTestId="restore-modal"
      closeModal={closeModal}
      headerMessage={
        isNewClusterMode ? Messages.headerMessageCreate : Messages.headerMessage
      }
      schema={pitrSchema}
      submitting={restoringFromBackup || restoringFromPointInTime}
      defaultValues={{ ...defaultValues, backupName: backupName || '' }}
      onSubmit={({ backupName, backupType, pitrBackup }) => {
        let pointInTimeDate = '';
        let pitrBackupName = '';
        if (pitrData) {
          pitrBackupName = pitrData.latestBackupName;
        }

        if (pitrBackup) {
          const pitrDateObj = new Date(pitrBackup);
          pointInTimeDate = pitrDateObj.toISOString().split('.')[0] + 'Z';
        }
        if (isNewClusterMode) {
          closeModal();
          const selectedBackup = backups?.find(
            (backup) => backup.name === backupName
          );
          if (backupType === BackuptypeValues.fromBackup) {
            navigate('/databases/new', {
              state: {
                selectedDbCluster: cluster.metadata.name,
                backupName,
                namespace,
                backupStorageName: selectedBackup,
              },
            });
          } else {
            navigate('/databases/new', {
              state: {
                selectedDbCluster: cluster.metadata.name,
                backupName: pitrBackupName,
                namespace,
                backupStorageName: selectedBackup,
                pointInTimeDate: pointInTimeDate,
              },
            });
          }
        } else {
          if (backupType === BackuptypeValues.fromBackup) {
            restoreBackupFromBackup(
              { backupName, namespace },
              {
                onSuccess() {
                  closeModal();
                  navigate('/');
                },
              }
            );
          } else {
            restoreBackupFromPointInTime(
              {
                backupName: pitrBackupName,
                namespace,
                pointInTimeDate: pointInTimeDate,
              },
              {
                onSuccess() {
                  closeModal();
                  navigate('/');
                },
              }
            );
          }
        }
      }}
      submitMessage={isNewClusterMode ? Messages.create : Messages.restore}
    >
      <ModalContent
        isLoading={isLoading}
        header={isNewClusterMode ? Messages.subHeadCreate : Messages.subHead}
        dbType={dbEngineToDbType(cluster.spec.engine.type)}
        pitrData={pitrData}
        backups={backups}
        backupStorageName={cluster.spec.backup?.pitr?.backupStorageName}
      />
    </FormDialog>
  );
};

export default RestoreDbModal;
