export const Messages = {
  dataImporter: {
    label: 'Select data Importer',
    placeholder: 'Select importer',
  },
  s3Details: {
    label: 'Provide S3 details',
    dialogTitle: 'S3 details',
    bucketName: 'Bucket name',
    bucketNamePlaceholder: 'Enter bucket name',
    warning:
      'OpenEverest does not validate S3 credentials or file paths. Please ensure your bucket, region, and paths are correct and accessible before proceeding.',

    region: 'Region',
    regionPlaceholder: 'Enter region',

    endpoint: 'Endpoint',
    endpointPlaceHolder: 'Enter URL',

    accessKey: 'Access key',
    accessKeyPlaceholder: 'Enter Access key',

    secretKey: 'Secret Key',
    secretKeyPlaceholder: 'Enter Secret key',

    verifyTLS: 'Verify TLS certificate',
    verifyTLSTooltip:
      "TLS verifies the server's certificate chain and host name, ensuring its authenticity.",

    forcePathStyle: 'Force path-style URL access',
    forcePathStyleTooltip:
      "Some storage providers require path-style URLs to access the bucket. Read the provider's documentation to determine if this is required.",
  },
  fileDir: {
    label: 'File Directory',
    dialogTitle: 'File directory',

    filePath: 'File path',
    filePathPlaceholder: 'Input file path',
  },
  dbCreds: {
    label: 'DB credentials',
    dialogTitle: 'DB credentials',

    root: 'root',
    proxyAdmin: 'proxyadmin',
    xtraBackup: 'xtrabackup',
    monitor: 'monitor',
    pmmServerPassword: 'PMM server password',
    operatorAdmin: 'Operator admin',
    replication: 'Replication',
  },
  config: {
    label: 'Configuration',

    recoveryTarget: 'Recovery target',
    recoveryTargetLSN: 'Recovery target lsn',
    recoveryTargetXID: 'Recovery target xid',
    recoveryTargetTime: 'Recovery target time',
    recoveryTargetName: 'Recovery target name',
  },
  fillDetails: 'Fill details',
  save: 'Save',
  enterPassword: 'Enter password',
  optional: '(optional)',
};
