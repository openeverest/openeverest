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

// Formats a resource value for the preview, applying the shards multiplier the
// same way the limits do. Returns an empty string for NaN values.
export const formatResourceValue = (
  parsedResource: number,
  unit: string,
  shardMultiplier?: number
) =>
  Number.isNaN(parsedResource)
    ? ''
    : `${
        shardMultiplier
          ? (shardMultiplier * parsedResource).toFixed(2)
          : parsedResource.toFixed(2)
      } ${unit}`;
