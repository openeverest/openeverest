export const Messages = {
  storageLimitHelperText: (activeStorages: number, maxStorages: number) =>
    activeStorages >= maxStorages
      ? `Storage limit reached. You are using ${activeStorages} out of ${maxStorages} available storages.`
      : `You are currently using ${activeStorages} out of ${maxStorages} available storages.`,
  inUseLabel: '(in use)',
};
