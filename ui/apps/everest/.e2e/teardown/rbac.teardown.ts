import {
  restoreOldRBACPermissions,
  waitForRBACRestored,
} from '@e2e/utils/rbac-cmd-line';
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import { switchUser } from '@e2e/utils/user';
import { test as setup } from '@playwright/test';

setup('RBAC teardown', async ({ page }) => {
  const restoreStartedAt = Date.now();
  await restoreOldRBACPermissions();
  await switchUser(page, process.env.CI_USER, process.env.CI_PASSWORD);
  await waitForRBACRestored(await getTokenFromLocalStorage());
  setup.info().annotations.push({
    type: 'rbac-restore-applied-after-ms',
    description: String(Date.now() - restoreStartedAt),
  });
});
