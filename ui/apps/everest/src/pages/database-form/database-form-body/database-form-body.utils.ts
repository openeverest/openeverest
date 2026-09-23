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

interface SubmitDisabledInput {
  isSubmitting: boolean;
  presetPending: boolean;
  presetSelected: boolean;
  isValid: boolean;
}

// A one-click preset deploy is gated only by the preset resolving (the server
// validates the resolved spec), so form validity must NOT block it. Manual
// (no-preset) creation still requires a valid form.
export const isSubmitDisabled = ({
  isSubmitting,
  presetPending,
  presetSelected,
  isValid,
}: SubmitDisabledInput): boolean =>
  isSubmitting || presetPending || (!presetSelected && !isValid);
