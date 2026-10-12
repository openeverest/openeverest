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

import { useEffect, useRef } from 'react';
import { useFormContext } from 'react-hook-form';
import { useFetchSecrets } from 'hooks/api/secrets';
import { useClusterName } from 'hooks/api/useClusterName';
import { getEditableSecretFields } from './secret-field.utils';
import { SecretNamespaceSyncProps } from './secret-field.types';

// A wizard mounts one step at a time, so its secret fields miss a namespace
// switch: this clears the picks the new namespace doesn't have.
export const SecretNamespaceSync = ({
  sections,
  namespace,
  provider,
}: SecretNamespaceSyncProps) => {
  const { getValues, setValue } = useFormContext();
  const cluster = useClusterName();
  const fetchSecrets = useFetchSecrets(cluster);
  const namespaceRef = useRef(namespace);

  useEffect(() => {
    const previousNamespace = namespaceRef.current;
    namespaceRef.current = namespace;
    if (
      !previousNamespace ||
      !namespace ||
      !provider ||
      namespace === previousNamespace
    ) {
      return;
    }

    getEditableSecretFields(sections).forEach(({ path, definition }) => {
      const pick: unknown = getValues(path);
      if (typeof pick !== 'string' || !pick) return;

      fetchSecrets(namespace, { provider, definition })
        .then((secrets) => {
          const superseded =
            namespaceRef.current !== namespace || getValues(path) !== pick;
          const exists = secrets.some(
            ({ metadata }) => metadata?.name === pick
          );
          if (!superseded && !exists) {
            setValue(path, '', { shouldValidate: true });
          }
        })
        // A pick that can't be checked stays.
        .catch(() => undefined);
    });
  }, [namespace, provider, sections, getValues, setValue, fetchSecrets]);

  return null;
};
