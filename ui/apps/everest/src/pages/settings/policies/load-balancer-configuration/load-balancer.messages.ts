export const messages = {
  title: 'LoadBalancer Configuration',
  rowActions: {
    viewDetails: 'View details',
    delete: 'Delete',
  },
  deleteDialog: {
    alertMessage: {
      inUse:
        'This config is currently in use by one or more clusters. Please unassign it first before deleting.',
      notInUse: 'This action will permanently delete your config.',
    },
    dialogContent: {
      firstPart: 'Are you sure you want to permanently delete ',
      secondPart:
        ' config? To confirm this action, type the name of your config.',
    },
  },
  configurationList: {
    emptyState: {
      contentSlot: 'You currently do not have any config',
    },
    createButton: 'Create configuration',
  },
  details: {
    editButton: 'Edit configuration',
    saveButton: 'Save configuration',
  },
};
