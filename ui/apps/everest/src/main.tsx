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

/* eslint-disable no-console */
// Must run before any plugin bundle is dynamically imported.
import './plugin-runtime-host';
import ReactDOM from 'react-dom/client';
import App from 'App';

// We don't use SSR, so we suppress this SSR annoying waring about first-child
const consoleError = console.error;

console.error = function filterErrors(msg, ...args) {
  if (/server-side rendering/.test(msg)) {
    return;
  }
  consoleError(msg, ...args);
};

ReactDOM.createRoot(document.getElementById('root')!).render(<App />);
