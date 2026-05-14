export const Messages = {
  storageLimitHelperText: (usedStorages: number, maxStorages: number) =>
    usedStorages >= maxStorages
      ? `Use an existing storage for this schedule, as you've already used the allowed storage limit (${maxStorages})`
      : `You are currently using ${usedStorages} out of ${maxStorages} available storages.`,
};
