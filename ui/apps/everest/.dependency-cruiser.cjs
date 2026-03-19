module.exports = {
  forbidden: [
    {
      name: 'no-circular',
      severity: 'error',
      comment:
        'Prevent circular dependencies in database-form and ui-generator.',
      from: {},
      to: {
        circular: true,
      },
    },
  ],
  options: {
    tsConfig: {
      fileName: './tsconfig.json',
    },
    includeOnly: '^src/(pages/database-form|components/ui-generator)',
    doNotFollow: {
      path: 'node_modules',
    },
    reporterOptions: {
      dot: {
        collapsePattern: 'node_modules/[^/]+',
      },
    },
  },
};
