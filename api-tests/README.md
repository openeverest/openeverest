# Percona everest API integration tests

Before running tests you need to [run Everest locally](../CONTRIBUTING.md#run-everest-locally).

## Running integration tests
There are several ways running tests
```
  npx playwright test
    Runs the end-to-end tests.

  npx playwright test tests/test.spec.ts
    Runs the specific tests.

  npx playwright test --ui
    Starts the interactive UI mode.

  npx playwright test --debug
    Runs the tests in debug mode.

  npx playwright codegen
    Auto generate tests with Codegen.
```

or
```bash
   make test
```

## CLI commands

There is a `cli` entity which is accessible during test and allows executing CLI commands on the host

Example usage:
```javascript

import { test, expect } from '@fixtures';

test('check mongodb-operator is installed', async ({ cli }) => {
    const output = await cli.execSilent('kubectl get pods --namespace=everest');
    await output.assertSuccess();
    await output.outContains('mongodb-operator');
});
```
