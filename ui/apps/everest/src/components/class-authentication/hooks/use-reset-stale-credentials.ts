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
import {
  getSectionFormDefaults,
  SectionForm,
} from 'components/ui-generator/utils/section-form';
import { AuthenticationFormFields } from '../class-authentication.constants';
import { AuthenticationFormValues } from '../class-authentication.types';

export interface CredentialsScope {
  namespace: string;
  className: string;
  authMode: string;
  // The class's credentials form for the mode.
  form?: SectionForm;
}

// What was picked or typed for another namespace, class or mode doesn't
// apply: existing Secrets are listed per namespace, class and mode, and the
// credentials are those the class's form asks for the mode.
export const useResetStaleCredentials = ({
  namespace,
  className,
  authMode,
  form,
}: CredentialsScope) => {
  const { setValue } = useFormContext<AuthenticationFormValues>();

  const secretListScope = JSON.stringify([namespace, className, authMode]);
  const previousSecretListScope = useRef(secretListScope);
  useEffect(() => {
    if (previousSecretListScope.current === secretListScope) {
      return;
    }
    previousSecretListScope.current = secretListScope;
    // Validated, so that an Existing source asks for a new pick at once.
    setValue(AuthenticationFormFields.existingSecretName, '', {
      shouldValidate: true,
    });
  }, [secretListScope, setValue]);

  const credentialsFormScope = JSON.stringify([className, authMode]);
  const previousCredentialsFormScope = useRef(credentialsFormScope);
  useEffect(() => {
    if (previousCredentialsFormScope.current === credentialsFormScope) {
      return;
    }
    previousCredentialsFormScope.current = credentialsFormScope;
    setValue(
      AuthenticationFormFields.credentials,
      form ? getSectionFormDefaults(form) : {}
    );
  }, [credentialsFormScope, form, setValue]);
};
