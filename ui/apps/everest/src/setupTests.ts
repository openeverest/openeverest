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

import '@testing-library/jest-dom';

// Disable MUI CSS transitions in jsdom so that popover/menu rendering is
// instant and deterministic.  jsdom has no CSS engine, so `transitionend`
// events never fire.  Under heavy CPU load the MUI fallback setTimeout
// (225 ms) can be starved and tests time out waiting for the popover content
// to become accessible.  Zeroing out every duration removes that dependency.
const style = document.createElement('style');
style.textContent =
  '*, *::before, *::after { transition-duration: 0s !important; animation-duration: 0s !important; }';
document.head.appendChild(style);
