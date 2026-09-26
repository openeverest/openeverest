#!/usr/bin/env node
// Serves a built host (ui/apps/everest/dist) the way the Go server does — nonce'd
// index template + production CSP — and fakes just enough of the API to log in
// and load plugin bundles from .variants/, so no backend/Kubernetes is needed.
//
// Env: PORT (default 4173), HOST_DIST (default ../apps/everest/dist),
//      EVEREST_VERSION (default 1.99.0), COMPAT_LOG_UNMOCKED=1 to log stubbed calls.
// Plugins to advertise come from the `compat_plugins` cookie: a JSON array of
// { name, variant, compatibleUiContractVersions?, compatibleHostVersions? }.
import crypto from 'node:crypto';
import fs from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const PORT = Number(process.env.PORT ?? 4173);
const HOST_DIST = path.resolve(HERE, process.env.HOST_DIST ?? '../apps/everest/dist');
const VARIANTS_DIR = path.join(HERE, '.variants');
const EVEREST_VERSION = process.env.EVEREST_VERSION ?? '1.99.0';
const CLUSTER = 'main';
const INDEX_TEMPLATE = fs.readFileSync(path.join(HOST_DIST, 'index.html'), 'utf8');
const HUB_CATALOG = fs.readFileSync(path.join(HERE, 'fixtures/hub-catalog.json'), 'utf8');

const MIME = {
  '.js': 'text/javascript',
  '.mjs': 'text/javascript',
  '.css': 'text/css',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.json': 'application/json',
  '.map': 'application/json',
  '.woff2': 'font/woff2',
  '.ttf': 'font/ttf',
};

// Mirrors securityHeaders() in internal/server/middlewares.go (non-OIDC, non-TLS).
function securityHeaders(nonce) {
  const n = `'nonce-${nonce}'`;
  return {
    'Content-Security-Policy': [
      "default-src 'self'",
      "font-src 'self' data:",
      "img-src 'self' data:",
      `script-src 'self' ${n}`,
      `style-src 'self' ${n} 'sha256-47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU='`,
      "form-action 'self'",
      "base-uri 'self'",
      "object-src 'none'",
      "frame-ancestors 'none'",
      "connect-src 'self'",
    ].join('; '),
    'Cross-Origin-Embedder-Policy': 'require-corp',
    'Cross-Origin-Opener-Policy': 'same-origin',
    'Cross-Origin-Resource-Policy': 'same-origin',
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': 'DENY',
    'Referrer-Policy': 'no-referrer',
    'Cache-Control': 'no-store, max-age=0',
  };
}

const b64url = (obj) => Buffer.from(JSON.stringify(obj)).toString('base64url');
function fakeAccessToken() {
  const now = Math.floor(Date.now() / 1000);
  return `${b64url({ alg: 'HS256', typ: 'JWT' })}.${b64url({ iss: 'everest', sub: 'admin:login', iat: now, exp: now + 3600 })}.c2ln`;
}

function cookie(req, name) {
  const raw = (req.headers.cookie ?? '').split(/;\s*/).find((c) => c.startsWith(`${name}=`));
  return raw ? decodeURIComponent(raw.slice(name.length + 1)) : undefined;
}

function advertisedPlugins(req) {
  try {
    return JSON.parse(cookie(req, 'compat_plugins') ?? '[]');
  } catch {
    return [];
  }
}

function send(res, status, body, headers = {}) {
  res.writeHead(status, headers);
  res.end(body);
}
const json = (res, body, status = 200) => send(res, status, JSON.stringify(body), { 'Content-Type': 'application/json' });

function sendFile(res, file, extraHeaders = {}) {
  if (!file || !fs.existsSync(file) || !fs.statSync(file).isFile()) return send(res, 404, 'not found');
  send(res, 200, fs.readFileSync(file), {
    'Content-Type': MIME[path.extname(file)] ?? 'application/octet-stream',
    'Cross-Origin-Resource-Policy': 'same-origin',
    ...extraHeaders,
  });
}

// Resolves `base/rel` and refuses anything that escapes `base`.
function safeJoin(base, rel) {
  const full = path.resolve(base, `.${path.posix.normalize(`/${rel}`)}`);
  return full.startsWith(base + path.sep) ? full : undefined;
}

// Minimal answers for the calls the host makes while booting; everything else gets [].
const API_STUBS = {
  'GET /v1/version': { projectUrl: '', version: EVEREST_VERSION, fullCommit: 'compat' },
  'GET /v1/settings': { oidcConfig: null },
  'GET /v1/namespaces': [],
  'GET /v1/session': {},
  'GET /v1/permissions': { enabled: false, permissions: [['*', '*', '*']] },
  'GET /v1/cluster-info': { clusterType: 'generic', storageClassNames: ['standard'] },
};

function handleApi(req, res, url) {
  const key = `${req.method} ${url.pathname}`;
  if (key === 'POST /v1/auth/token') {
    return json(res, { access_token: fakeAccessToken(), token_type: 'Bearer', expires_in: 3600 });
  }

  const pluginsBase = `/v1/clusters/${CLUSTER}/plugins`;
  if (key === `GET ${pluginsBase}`) {
    return json(
      res,
      advertisedPlugins(req).map((p) => ({
        name: p.name,
        displayName: p.name,
        bundleUrl: `${pluginsBase}/${p.name}/main.js?variant=${encodeURIComponent(p.variant)}`,
        compatibleHostVersions: p.compatibleHostVersions,
        compatibleUiContractVersions: p.compatibleUiContractVersions,
      })),
    );
  }
  if (url.pathname.startsWith(`${pluginsBase}/`)) {
    const [name, ...rest] = url.pathname.slice(pluginsBase.length + 1).split('/');
    const sub = rest.join('/');
    if (sub === 'main.js' || sub === 'main.js.map') {
      const variant = url.searchParams.get('variant') ?? advertisedPlugins(req).find((p) => p.name === name)?.variant;
      const dir = variant ? safeJoin(VARIANTS_DIR, `${name}/${variant}`) : undefined;
      return sendFile(res, dir && path.join(dir, sub));
    }
    if (sub === 'api/catalog' || sub === 'api/summary') return send(res, 200, HUB_CATALOG, { 'Content-Type': 'application/json' });
    if (sub === 'api/installed') return json(res, { items: [{ name: 'percona-postgresql', type: 'provider', version: '2.0.0' }] });
    return send(res, 404, 'not found');
  }

  if (key in API_STUBS) return json(res, API_STUBS[key]);
  if (process.env.COMPAT_LOG_UNMOCKED) console.log(`[compat] unmocked ${key}`);
  return json(res, req.method === 'GET' ? [] : {});
}

http
  .createServer((req, res) => {
    const url = new URL(req.url ?? '/', `http://${req.headers.host}`);
    if (url.pathname.startsWith('/v1/')) return handleApi(req, res, url);
    if (url.pathname.startsWith('/static/')) {
      return sendFile(res, safeJoin(HOST_DIST, url.pathname.slice(1)), securityHeaders('static'));
    }
    const nonce = crypto.randomBytes(16).toString('base64');
    const html = INDEX_TEMPLATE.replaceAll('{{.CSPNonce}}', nonce).replaceAll('{{.EverestVersion}}', EVEREST_VERSION);
    return send(res, 200, html, { 'Content-Type': 'text/html; charset=utf-8', ...securityHeaders(nonce) });
  })
  .listen(PORT, () => console.log(`[compat] host ${HOST_DIST} on http://localhost:${PORT} (version ${EVEREST_VERSION})`));
