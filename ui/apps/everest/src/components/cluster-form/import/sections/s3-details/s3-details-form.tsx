import { Alert } from '@mui/material';
import { TextInput } from '@percona/ui-lib';
import { ImportFields } from '../../import.types';
import { Messages } from '../../messages';
import { S3Checkbox } from './s3-checkbox';
import { HiddenInput } from 'components/hidden-input';

export const S3DetailsForm = () => {
  return (
    <>
      <Alert severity="warning" sx={{ mb: 2 }}>
        {Messages.s3Details.warning}
      </Alert>
      <TextInput
        name={ImportFields.bucketName}
        label={Messages.s3Details.bucketName}
        textFieldProps={{
          placeholder: Messages.s3Details.bucketNamePlaceholder,
        }}
        isRequired
      />
      <TextInput
        name={ImportFields.region}
        label={Messages.s3Details.region}
        textFieldProps={{
          placeholder: Messages.s3Details.regionPlaceholder,
        }}
        isRequired
      />

      <TextInput
        name={ImportFields.endpoint}
        label={Messages.s3Details.endpoint}
        textFieldProps={{
          placeholder: Messages.s3Details.endpointPlaceHolder,
        }}
        isRequired
      />
      <HiddenInput
        name={ImportFields.accessKey}
        label={Messages.s3Details.accessKey}
        textFieldProps={{
          placeholder: Messages.s3Details.accessKeyPlaceholder,
        }}
        isRequired
      />
      <HiddenInput
        name={ImportFields.secretKey}
        label={Messages.s3Details.secretKey}
        textFieldProps={{
          placeholder: Messages.s3Details.secretKeyPlaceholder,
        }}
        isRequired
      />

      <S3Checkbox
        name={ImportFields.verifyTlS}
        text={Messages.s3Details.verifyTLS}
        tooltip={Messages.s3Details.verifyTLSTooltip}
      />
      <S3Checkbox
        name={ImportFields.forcePathStyle}
        text={Messages.s3Details.forcePathStyle}
        tooltip={Messages.s3Details.forcePathStyleTooltip}
      />
    </>
  );
};
