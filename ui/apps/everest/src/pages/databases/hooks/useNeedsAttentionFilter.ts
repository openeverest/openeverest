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

import { useEffect, useMemo, useState } from 'react';
import { InstanceTableElement } from '../dbClusterView.types';
import { doesRowNeedAttention } from '../DbClusterView.utils';

// Narrows the list to the Instances someone has to look at.
export const useNeedsAttentionFilter = (rows: InstanceTableElement[]) => {
  const [selected, setSelected] = useState(false);
  const rowsNeedingAttention = useMemo(
    () => rows.filter(doesRowNeedAttention),
    [rows]
  );
  const count = rowsNeedingAttention.length;

  // Once nothing needs attention the filter would only hide every row, and it
  // should not come back on by itself with the next problem.
  useEffect(() => {
    if (count === 0) {
      setSelected(false);
    }
  }, [count]);

  const active = selected && count > 0;

  return {
    count,
    active,
    visibleRows: active ? rowsNeedingAttention : rows,
    toggle: () => setSelected((value) => !value),
  };
};
