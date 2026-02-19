import { test as teardown } from '@playwright/test';
import { deleteDbClusterFn } from '@e2e/utils/db-cluster';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import { dbClusterName } from './project.config';

teardown.describe.serial('DB Cluster Restore Action teardown', () => {
  teardown(`Delete ${dbClusterName} cluster`, async ({ request }) => {
    await deleteDbClusterFn(
      request,
      dbClusterName,
      EVEREST_CI_NAMESPACES.EVEREST_UI
    );
  });
});
