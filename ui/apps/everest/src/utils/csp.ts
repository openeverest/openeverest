// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/**
 * Extracts the CSP nonce from the meta tag injected by the Go server.
 *
 * The server renders `<meta name="csp-nonce" content="{{.CSPNonce}}" />`
 * into index.html on every request. This nonce is consumed by:
 * - Emotion cache (for MUI styled components)
 * - CodeMirror 6 (via EditorView.cspNonce facet)
 *
 * Returns an empty string in dev mode (Vite dev server) where no
 * server-side nonce injection occurs.
 */
export const getCspNonce = (): string =>
  document
    .querySelector('meta[name="csp-nonce"]')
    ?.getAttribute('content') ?? '';
