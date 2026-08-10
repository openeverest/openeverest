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

import { useLocation } from 'react-router-dom';
import {
  parseRestoreNavigationRouteState,
  ParsedRestoreNavigationState,
} from '../restore-navigation-state';

// Single reader + validator for the restore-to-new-DB payload the modal passes
// to the wizard via router state. Centralizes the (previously triplicated) shape
// checks so mode detection, schema resolution, and prefill stay in agreement,
// and validates the untyped router state before it reaches the restore payload.
export const useRestoreNavigationState = (): RestoreNavigationState => {
  const { state } = useLocation();
  return parseRestoreNavigationRouteState(state);
};

export type RestoreNavigationState = ParsedRestoreNavigationState;
