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

import { useEffect, useState } from 'react';
import { useMonitoringDestination } from 'hooks/api/monitoring-destinations';
import { MonitoringDestinationForm } from './monitoring-destination-form/monitoring-destination-form';
import { MonitoringDestinationModalProps } from './monitoring-destination-modal.types';

// Edits start from the destination as the server has it now, so the save
// carries its current resourceVersion.
export const MonitoringDestinationModal = ({
  target,
  classes,
  namespaces,
  onClose,
}: MonitoringDestinationModalProps) => {
  const [changedMeanwhile, setChangedMeanwhile] = useState(false);
  const {
    data: original,
    refetch,
    isError,
  } = useMonitoringDestination(target?.namespace ?? '', target?.name ?? '', {
    enabled: !!target,
    staleTime: 0,
    gcTime: 0,
  });

  // The request already reported the error; don't leave an invisible modal open.
  useEffect(() => {
    if (isError && !original) {
      onClose();
    }
  }, [isError, original, onClose]);

  if (target && !original) {
    return null;
  }

  return (
    <MonitoringDestinationForm
      key={original?.metadata?.resourceVersion}
      original={original}
      classes={classes}
      namespaces={namespaces}
      changedMeanwhile={changedMeanwhile}
      onChangedMeanwhile={() => {
        setChangedMeanwhile(true);
        refetch();
      }}
      onClose={onClose}
    />
  );
};
