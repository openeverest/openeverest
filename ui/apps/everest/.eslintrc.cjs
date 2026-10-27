module.exports = {
  root: true,
  extends: ['@percona/eslint-config-react'],
  rules: {
    'no-restricted-imports': [
      'error',
      {
        patterns: [
          {
            group: ['**/utils/rbac', 'utils/rbac', './rbac'],
            importNames: ['can', 'canAll', 'cannot'],
            message:
              'Check permissions through the hooks in hooks/rbac: they build the RBAC object for the resource.',
          },
        ],
      },
    ],
  },
  overrides: [
    {
      files: ['src/hooks/rbac/**', 'src/utils/rbac.test.ts'],
      rules: { 'no-restricted-imports': 'off' },
    },
  ],
};
