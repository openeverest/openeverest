module.exports = {
  root: true,
  extends: ['@percona/eslint-config-react', 'plugin:storybook/recommended'],
  plugins: ['import'],
  settings: {
    'import/resolver': {
      typescript: {
        project: './tsconfig.json',
      },
    },
  },
  rules: {
    // Prevent internal ui-lib components from importing through the package root barrel.
    // Use direct relative paths (e.g. '../../../labeled-content') instead.
    'import/no-restricted-paths': [
      'error',
      {
        zones: [
          {
            target: './src/**',
            from: './src/index.ts',
            message:
              "Do not import from the ui-lib package root barrel inside src/. Use a direct relative path (e.g. '../../../labeled-content') instead.",
          },
        ],
      },
    ],
    'no-restricted-imports': [
      'error',
      {
        patterns: [
          {
            group: ['@percona/ui-lib', '@percona/ui-lib/*'],
            message:
              'Do not import from the package itself inside src/. Use a direct relative path instead.',
          },
        ],
      },
    ],
  },
};
