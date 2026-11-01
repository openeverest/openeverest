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
import { SwitchInput } from '@percona/ui-lib';
import { useWatch } from 'react-hook-form';
import { TOGGLEABLE_SWITCHES_KEY } from 'components/ui-generator/utils/toggleable/toggleable';
import { joinPath } from 'components/ui-generator/utils/object-path';
import { useUiGeneratorContext } from 'components/ui-generator/ui-generator-context';
import { BorderedWrapper } from '../bordered-wrapper';
import { ToggleableWrapperProps } from './toggleable-wrapper.types';
import { Messages } from './toggleable-wrapper.messages';
import { FOCUSABLE_FIELD } from './toggleable-wrapper.constants';

// A bordered card whose heading switch shows or hides its fields.
export const ToggleableWrapper = ({
  label,
  description,
  toggleable,
  children,
}: ToggleableWrapperProps) => {
  const { root } = useUiGeneratorContext();
  const switchName = toggleable && joinPath(root, toggleable.switchName);
  const bodyRef = useRef<HTMLDivElement>(null);
  const enabledByUser = useRef(false);
  // Watched unconditionally (hook rules); the fallback name is never read.
  const switchValue = useWatch({ name: switchName ?? TOGGLEABLE_SWITCHES_KEY });
  const isOn = !switchName || switchValue === true;

  useEffect(() => {
    if (import.meta.env.DEV && !switchName) {
      // eslint-disable-next-line no-console
      console.warn(
        `[ui-generator] toggleable group "${label ?? ''}" rendered without preprocessSchema; it behaves as a bordered group`
      );
    }
  }, [switchName, label]);

  // Untouched fields are not flagged: errors appear on input, like elsewhere in the form.
  useEffect(() => {
    if (!isOn || !enabledByUser.current) return;
    enabledByUser.current = false;
    bodyRef.current?.querySelector<HTMLElement>(FOCUSABLE_FIELD)?.focus();
  }, [isOn]);

  return (
    <BorderedWrapper
      label={label}
      description={description}
      bodyRef={bodyRef}
      action={
        switchName && (
          <SwitchInput
            name={switchName}
            label={Messages.switchLabel}
            // Forms whose defaults were built before the schema (e.g. live
            // schema editing) have no switch value; keep the Switch controlled.
            controllerProps={{ name: switchName, defaultValue: false }}
            switchFieldProps={{
              onChange: (_event, checked) => {
                enabledByUser.current = checked;
              },
              slotProps: {
                input: { 'aria-label': Messages.switchAriaLabel(label) },
              },
            }}
          />
        )
      }
    >
      {isOn ? children : null}
    </BorderedWrapper>
  );
};
