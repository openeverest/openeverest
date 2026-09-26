// Host/plugin UI independence suite (PR #3076): every prebuilt plugin bundle in
// .variants/ must load, stay themed by the host tokens and leave the host's own
// styling untouched — whatever MUI version the plugin or the host was built with.
import { expect, test, type Page } from '@playwright/test';
import {
  collectCspViolations,
  emotionKeys,
  hostStyleSnapshot,
  listVariants,
  normalizeFont,
  openHost,
  rawToken,
  resolveToken,
  rootTokens,
  toggleHostDarkMode,
  unscopedPluginRules,
  type PageIssues,
  type Variant,
} from './harness';

const ROOT_TEST_ID: Record<string, string> = {
  canary: 'canary-root',
  'plugin-hub': 'plugin-hub-root',
};
const PRIMARY_BUTTON_TEST_ID: Record<string, string> = {
  canary: 'canary-button',
  'plugin-hub': 'plugin-hub-add-extension',
};

async function expectClean(page: Page, issues: PageIssues) {
  await collectCspViolations(page, issues);
  expect(issues.errors, 'console errors / page errors').toEqual([]);
  expect(issues.cspViolations, 'CSP violations').toEqual([]);
}

async function openPlugin(page: Page, context: import('@playwright/test').BrowserContext, baseURL: string, v: Variant, colorMode: 'light' | 'dark' = 'light') {
  const issues = await openHost(context, page, {
    baseURL,
    plugins: [{ name: v.plugin, variant: v.variant }],
    path: `/plugins/${v.plugin}`,
    colorMode,
  });
  await expect(page.getByTestId(ROOT_TEST_ID[v.plugin])).toBeVisible();
  return issues;
}

async function expectThemedLike(page: Page, testId: string, mode: 'light' | 'dark') {
  const primary = await resolveToken(page, '--everest-color-primary-main', 'background-color');
  const button = page.getByTestId(testId);
  await expect(button, `${testId} background follows host primary (${mode})`).toHaveCSS('background-color', primary);
  const transform = await rawToken(page, '--everest-font-button-transform');
  if (transform) await expect(button).toHaveCSS('text-transform', transform);
}

for (const v of listVariants()) {
  test.describe(`${v.plugin} @ ${v.variant}`, () => {
    test('loads without errors and registers its route', async ({ page, context, baseURL }) => {
      const issues = await openPlugin(page, context, baseURL!, v);
      const label = v.plugin === 'canary' ? 'Canary' : 'Plugin Hub';
      await expect(page.getByRole('listitem').getByRole('button', { name: label, exact: true })).toBeVisible();
      await expectClean(page, issues);
    });

    test('shares the host React (hooks and state work)', async ({ page, context, baseURL }) => {
      const issues = await openPlugin(page, context, baseURL!, v);
      if (v.plugin === 'canary') {
        await page.getByTestId('canary-counter').click();
        await page.getByTestId('canary-counter').click();
        await expect(page.getByTestId('canary-counter')).toHaveText('Clicked 2');
        await page.getByRole('textbox', { name: 'Echo' }).fill('hello');
        await expect(page.getByTestId('canary-echo')).toHaveText('hello');
        const probe = await page.evaluate(() => (window as unknown as { __pluginProbe__: Record<string, { reactIsHost: boolean; muiVersion: string }> }).__pluginProbe__.canary);
        expect(probe.reactIsHost).toBe(true);
      } else {
        await expect(page.getByRole('row')).toHaveCount(4);
        await page.getByPlaceholder(/Search by name/).fill('inspector');
        await expect(page.getByRole('row')).toHaveCount(2);
      }
      await expectClean(page, issues);
    });

    if (v.plugin === 'canary') {
      test('runs the MUI it was built with, not the host copy', async ({ page, context, baseURL }) => {
        await openPlugin(page, context, baseURL!, v);
        await expect(page.getByTestId('canary-body')).toHaveText(`Plugin running MUI ${v.requestedMui}`);
      });
    }

    test('inherits host palette, typography and shape', async ({ page, context, baseURL }) => {
      const issues = await openPlugin(page, context, baseURL!, v);
      await expectThemedLike(page, PRIMARY_BUTTON_TEST_ID[v.plugin], 'light');

      const hostBodyFont = normalizeFont(await rawToken(page, '--everest-font-body1-family'));
      const textTestId = v.plugin === 'canary' ? 'canary-body' : 'plugin-hub-root';
      const pluginFont = normalizeFont(await page.getByTestId(textTestId).evaluate((el) => getComputedStyle(el).fontFamily));
      expect(pluginFont, 'body1 font family').toBe(hostBodyFont);

      if (v.plugin === 'canary') {
        const radius = parseFloat(await rawToken(page, '--everest-radius'));
        const paperRadius = await page.getByTestId('canary-paper').evaluate((el) => parseFloat(getComputedStyle(el).borderTopLeftRadius));
        expect(paperRadius, 'Paper radius follows --everest-radius').toBe(radius);
        // A component imported straight from @mui/material must be themed too.
        await expectThemedLike(page, 'canary-direct-button', 'light');
      }
      await expectClean(page, issues);
    });

    test('follows host dark mode live', async ({ page, context, baseURL }) => {
      const issues = await openPlugin(page, context, baseURL!, v);
      const surfaceTestId = 'canary-paper';
      const lightPaper = await resolveToken(page, '--everest-color-background-paper', 'background-color');

      await toggleHostDarkMode(page);
      await expect(page.locator('html')).toHaveAttribute('data-everest-color-scheme', 'dark');
      const darkPaper = await resolveToken(page, '--everest-color-background-paper', 'background-color');
      expect(darkPaper).not.toBe(lightPaper);
      await expectThemedLike(page, PRIMARY_BUTTON_TEST_ID[v.plugin], 'dark');
      if (v.plugin === 'canary') {
        await expect(page.getByTestId(surfaceTestId)).toHaveCSS('background-color', darkPaper);
      }
      const darkText = await resolveToken(page, '--everest-color-text-primary');
      await expect(page.getByTestId(v.plugin === 'canary' ? 'canary-body' : 'plugin-hub-root').first()).toHaveCSS('color', darkText);
      await expectClean(page, issues);
    });

    test('portals and modals are themed and clean up', async ({ page, context, baseURL }) => {
      const issues = await openPlugin(page, context, baseURL!, v);
      const paper = await resolveToken(page, '--everest-color-background-paper', 'background-color');
      const bodyOverflowBefore = await page.evaluate(() => document.body.style.overflow);

      if (v.plugin === 'canary') {
        await page.getByTestId('canary-dialog-open').click();
        const dialog = page.locator('.MuiDialog-paper').filter({ hasText: 'Canary dialog' });
        await expect(dialog).toBeVisible();
        await expect(dialog).toHaveCSS('background-color', paper);
        await page.getByTestId('canary-dialog-close').click();
        await expect(dialog).toBeHidden();

        await page.getByTestId('canary-menu-open').click();
        await expect(page.getByTestId('canary-menu-item')).toBeVisible();
        await page.getByTestId('canary-menu-item').click();
        await expect(page.getByTestId('canary-menu-item')).toBeHidden();

        await page.getByTestId('canary-tooltip-anchor').hover();
        await expect(page.getByRole('tooltip')).toHaveText('Canary tooltip');
      } else {
        await page.getByRole('row', { name: /Inspector/ }).click();
        const drawer = page.locator('.MuiDrawer-paper').filter({ hasText: 'Install with Helm' });
        await expect(drawer).toBeVisible();
        await expect(drawer).toHaveCSS('background-color', paper);
        await page.getByRole('button', { name: 'Close' }).click();
        await expect(drawer).toBeHidden();
      }
      await expect.poll(() => page.evaluate(() => document.body.style.overflow), { message: 'body scroll lock released' }).toBe(bodyOverflowBefore);
      await expect(page.locator('[aria-hidden="true"]#root')).toHaveCount(0);
      await expectClean(page, issues);
    });

    test('styles are scoped to the plugin Emotion cache', async ({ page, context, baseURL }) => {
      const issues = await openPlugin(page, context, baseURL!, v);
      const keys = await emotionKeys(page);
      expect(keys, 'plugin injects its own Emotion cache').toContain(v.plugin);
      expect(await unscopedPluginRules(page, v.plugin), 'plugin rules not scoped to its cache key').toEqual([]);
      await expectClean(page, issues);
    });

    test('leaves host tokens and host chrome untouched', async ({ page, context, baseURL }) => {
      const baseline = await openHost(context, page, { baseURL: baseURL!, plugins: [], path: '/databases' });
      await expect(page.locator('header.MuiAppBar-root')).toBeVisible();
      const tokensBefore = await rootTokens(page);
      const hostBefore = await hostStyleSnapshot(page);
      expect(Object.keys(tokensBefore).length, 'host publishes tokens').toBeGreaterThan(20);
      await expectClean(page, baseline);

      await context.clearCookies();
      await context.addCookies([
        { name: 'compat_plugins', value: encodeURIComponent(JSON.stringify([{ name: v.plugin, variant: v.variant }])), url: baseURL! },
      ]);
      await page.goto(`/plugins/${v.plugin}`);
      await expect(page.getByTestId(ROOT_TEST_ID[v.plugin])).toBeVisible();
      // Back to a host page with the plugin's styles still in <head>.
      await page.getByRole('button', { name: 'Instances' }).click();
      await expect(page).toHaveURL(/\/databases/);
      await expect(page.locator('header.MuiAppBar-root')).toBeVisible();

      expect(await rootTokens(page)).toEqual(tokensBefore);
      expect(await hostStyleSnapshot(page)).toEqual(hostBefore);
    });
  });
}

test.describe('compatibility gates', () => {
  const baseline = listVariants().find((v) => v.plugin === 'canary' && v.variant === 'mui-7.3.11');
  test.skip(!baseline, 'needs the canary mui-7.3.11 variant');

  for (const range of ['>=19.0.0', '>=19']) {
    test(`skips a plugin whose UI contract (React major) does not match: ${range}`, async ({ page, context, baseURL }) => {
      // Major-only ranges fall through satisfiesRange()'s x.y.z regex and pass (plugin-version.ts).
      test.fail(range === '>=19', 'known bug: short-form ranges are not enforced');
      const issues = await openHost(context, page, {
        baseURL: baseURL!,
        plugins: [{ name: 'canary', variant: 'mui-7.3.11', compatibleUiContractVersions: range }],
        path: '/plugins/canary',
      });
      await expect(page.getByText('Plugin not found')).toBeVisible();
      await expect(page.getByTestId('canary-root')).toHaveCount(0);
      expect(issues.errors.join('\n')).toMatch(/Skipping "canary": needs UI contract/);
    });
  }

  test('skips a plugin that requires a newer host', async ({ page, context, baseURL }) => {
    const issues = await openHost(context, page, {
      baseURL: baseURL!,
      plugins: [{ name: 'canary', variant: 'mui-7.3.11', compatibleHostVersions: '>=2.0.0' }],
      path: '/plugins/canary',
    });
    await expect(page.getByText('Plugin not found')).toBeVisible();
    expect(issues.errors.join('\n')).toMatch(/Skipping "canary": requires host/);
  });
});
