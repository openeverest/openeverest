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

// By index, not by height: an item keeps its column when another one grows.
export const distributeIntoColumns = <T>(items: T[], count: number): T[][] => {
  const columns = Math.max(1, count);
  return Array.from({ length: columns }, (_, column) =>
    items.filter((_, index) => index % columns === column)
  );
};
