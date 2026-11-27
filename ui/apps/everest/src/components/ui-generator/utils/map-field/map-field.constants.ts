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

// What a row's field can't act on yet. A row starts empty and writes only its
// key and value, so a default and badges have nothing to act on.
export const IGNORED_ROW_FIELD_PARAMS = ['defaultValue', 'badge', 'badgeToApi'];
// A row can't evaluate CEL yet.
export const IGNORED_ROW_FIELD_RULES = ['celExpressions'];
// A row's field takes no options from an API yet, and is the same field in
// every form mode: one hidden in a mode would no longer hold the key or the
// value, and the map would be written without its entries.
export const IGNORED_ROW_FIELD_OPTIONS = ['dataSource', 'modes'];
