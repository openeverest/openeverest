module.exports = {
  root: true,
  extends: ['@percona/eslint-config-react', 'plugin:storybook/recommended'],
  rules: {
    // Prevent internal ui-lib components from importing through the package root barrel.
    // Use direct relative paths (e.g. '../../../labeled-content') instead.
    'no-restricted-imports': [
      'error',
      {
        paths: [
          {
            name: '../../..',
            message:
              "Do not import from the ui-lib package root barrel inside src/. Use a direct relative path (e.g. '../../../labeled-content') instead.",
          },
          {
            name: '../../../index',
            message:
              "Do not import from the ui-lib package root barrel inside src/. Use a direct relative path (e.g. '../../../labeled-content') instead.",
          },
        ],
      },
    ],
  },
};
