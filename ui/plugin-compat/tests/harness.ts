import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import type { BrowserContext, Page } from '@playwright/test';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const VARIANTS_DIR = path.join(HERE, '../.variants');

export interface Variant {
  plugin: string;
  variant: string;
  requestedMui: string;
}

export interface AdvertisedPlugin {
  name: string;
  variant: string;
  compatibleUiContractVersions?: string;
  compatibleHostVersions?: string;
}

// Every built bundle under .variants/<plugin>/<variant>, optionally narrowed by COMPAT_FILTER (regex on "plugin/variant").
export function listVariants(): Variant[] {
  const filter = process.env.COMPAT_FILTER ? new RegExp(process.env.COMPAT_FILTER) : undefined;
  if (!fs.existsSync(VARIANTS_DIR)) return [];
  return fs
    .readdirSync(VARIANTS_DIR, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .flatMap(({ name: plugin }) =>
      fs.readdirSync(path.join(VARIANTS_DIR, plugin)).map((variant) => {
        const meta = JSON.parse(fs.readFileSync(path.join(VARIANTS_DIR, plugin, variant, 'variant.json'), 'utf8'));
        return { plugin, variant, requestedMui: meta.requestedMui };
      }),
    )
    .filter((v) => !filter || filter.test(`${v.plugin}/${v.variant}`));
}

export interface PageIssues {
  errors: string[];
  cspViolations: string[];
}

// Host noise that is unrelated to plugins.
const IGNORED_ERRORS = [/React Router Future Flag/];

export async function openHost(
  context: BrowserContext,
  page: Page,
  opts: { baseURL: string; plugins: AdvertisedPlugin[]; path: string; colorMode?: 'light' | 'dark' },
): Promise<PageIssues> {
  await context.addCookies([
    { name: 'compat_plugins', value: encodeURIComponent(JSON.stringify(opts.plugins)), url: opts.baseURL },
  ]);
  await context.addInitScript((colorMode) => {
    localStorage.setItem('welcomeModal', 'false');
    localStorage.setItem('colorMode', colorMode);
    const violations: string[] = [];
    Object.defineProperty(window, '__cspViolations', { value: violations });
    document.addEventListener('securitypolicyviolation', (e) =>
      violations.push(`${e.violatedDirective} ${e.blockedURI} ${e.sourceFile}:${e.lineNumber}`),
    );
  }, opts.colorMode ?? 'light');

  const issues: PageIssues = { errors: [], cspViolations: [] };
  page.on('pageerror', (e) => issues.errors.push(`pageerror: ${e.message}`));
  page.on('response', (r) => {
    if (r.status() >= 400) issues.errors.push(`HTTP ${r.status()} ${new URL(r.url()).pathname}`);
  });
  page.on('console', (m) => {
    if (m.type() !== 'error' || IGNORED_ERRORS.some((re) => re.test(m.text()))) return;
    // Reported with its URL by the response listener above.
    if (m.text().startsWith('Failed to load resource')) return;
    issues.errors.push(m.text());
  });
  await page.goto(opts.path);
  return issues;
}

export async function collectCspViolations(page: Page, issues: PageIssues): Promise<void> {
  issues.cspViolations.push(...(await page.evaluate(() => (window as unknown as { __cspViolations: string[] }).__cspViolations)));
}

// Resolves a CSS custom property on :root to the computed value of `prop` (e.g. rgb(...)).
export function resolveToken(page: Page, token: string, prop = 'color'): Promise<string> {
  return page.evaluate(
    ([t, p]) => {
      const probe = document.createElement('div');
      probe.style.setProperty(p, `var(${t})`);
      document.body.appendChild(probe);
      const value = getComputedStyle(probe).getPropertyValue(p);
      probe.remove();
      return value;
    },
    [token, prop] as const,
  );
}

export function rawToken(page: Page, token: string): Promise<string> {
  return page.evaluate((t) => getComputedStyle(document.documentElement).getPropertyValue(t).trim(), token);
}

// All --everest-* and --mui-* custom properties on :root.
export function rootTokens(page: Page): Promise<Record<string, string>> {
  return page.evaluate(() => {
    const out: Record<string, string> = {};
    const cs = getComputedStyle(document.documentElement);
    for (let i = 0; i < cs.length; i++) {
      const name = cs[i];
      if (name.startsWith('--everest-') || name.startsWith('--mui-')) out[name] = cs.getPropertyValue(name).trim();
    }
    return out;
  });
}

const HOST_PROBES = {
  appBar: 'header.MuiAppBar-root',
  navDrawer: '.MuiDrawer-paper',
  userButton: '[data-testid="user-appbar-button"]',
  body: 'body',
  main: 'main',
};
const HOST_PROPS = ['color', 'background-color', 'font-family', 'font-size', 'border-radius', 'padding', 'box-shadow', 'text-transform'];

// Computed styles of fixed host chrome, to prove a plugin didn't restyle the host.
export function hostStyleSnapshot(page: Page): Promise<Record<string, Record<string, string> | null>> {
  return page.evaluate(
    ([probes, props]) =>
      Object.fromEntries(
        Object.entries(probes).map(([key, selector]) => {
          const el = document.querySelector(selector);
          if (!el) return [key, null];
          const cs = getComputedStyle(el);
          return [key, Object.fromEntries(props.map((p) => [p, cs.getPropertyValue(p)]))];
        }),
      ),
    [HOST_PROBES, HOST_PROPS] as const,
  );
}

// CSS rules in the plugin's Emotion sheets whose selector isn't scoped to its cache key.
export function unscopedPluginRules(page: Page, cacheKey: string): Promise<string[]> {
  return page.evaluate((key) => {
    const leaks: string[] = [];
    for (const style of document.querySelectorAll<HTMLStyleElement>(`style[data-emotion^="${key}"]`)) {
      const sheet = style.sheet;
      if (!sheet) continue;
      for (const rule of Array.from(sheet.cssRules)) {
        if (!(rule instanceof CSSStyleRule)) continue;
        const selectors = rule.selectorText.split(',').map((s) => s.trim());
        const scoped = selectors.every((s) => s.includes(`.${key}-`));
        if (!scoped) leaks.push(rule.selectorText);
      }
    }
    return leaks;
  }, cacheKey);
}

export function emotionKeys(page: Page): Promise<string[]> {
  return page.evaluate(() => [
    ...new Set(Array.from(document.querySelectorAll('style[data-emotion]')).map((s) => s.getAttribute('data-emotion')?.split(' ')[0] ?? '')),
  ]);
}

export const normalizeFont = (f: string) => f.replace(/["']/g, '').replace(/\s*,\s*/g, ',').trim().toLowerCase();

export async function toggleHostDarkMode(page: Page): Promise<void> {
  await page.getByTestId('user-appbar-button').click();
  await page.getByLabel('Dark mode').click();
  await page.keyboard.press('Escape');
  // The pointer would otherwise rest over plugin-hub's header button and apply its :hover color.
  await page.mouse.move(0, 400);
}
