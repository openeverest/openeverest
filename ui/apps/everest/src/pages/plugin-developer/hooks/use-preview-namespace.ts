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

import { useState, useEffect } from 'react';
import { useNamespaces } from 'hooks/api/namespaces';

type PreviewNamespace = {
  namespaces: string[];
  selectedNamespace: string;
  setSelectedNamespace: (namespace: string) => void;
  isLoading: boolean;
};

// Namespace context for the preview. Data-source providers (storage classes,
// monitoring configs) only fetch when a namespace is set, so we let the
// developer pick one and default to the first available.
export const usePreviewNamespace = (): PreviewNamespace => {
  const { data: namespaces = [], isLoading } = useNamespaces();
  const [selectedNamespace, setSelectedNamespace] = useState('');

  useEffect(() => {
    if (!selectedNamespace && namespaces.length > 0) {
      setSelectedNamespace(namespaces[0]);
    }
  }, [namespaces, selectedNamespace]);

  return { namespaces, selectedNamespace, setSelectedNamespace, isLoading };
};
