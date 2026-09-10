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

// Show only the most recent backups in the overview; the rest are reachable via
// the "See other X backups" link to the Backups tab. Kept below the ui-lib Table
// pagination threshold (10) so the bulky pager never renders here.
export const MAX_VISIBLE_BACKUPS = 5;
