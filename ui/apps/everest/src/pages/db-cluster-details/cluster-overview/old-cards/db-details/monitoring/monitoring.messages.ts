export const Messages = {
  editMonitoring: 'Edit Monitoring',
  addMonitoring: 'Add monitoring endpoint',
  alertText: (namespace: string) =>
    `Database monitoring is currently disabled because no monitoring endpoints have been configured. To enable database monitoring, first add a monitoring endpoint for the ${namespace} namespace`,
  monitoringInputLabel: 'Monitoring endpoint URL',
  enableMonitoring: 'Enable monitoring',
  save: 'Save',
};
