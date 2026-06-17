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

import { Provider } from 'shared-types/api.types';

export enum FormMode {
  New = 'new',
  Edit = 'edit',
  Restore = 'restore',
  Import = 'import',
}

export type ComponentModeOverrides = Partial<
  Record<FormMode, { uiType?: FieldType | 'hidden' }>
>;

export type FieldParamsModeOverrides = Partial<
  Record<
    FormMode,
    {
      disabled?: boolean;
      readOnly?: boolean;
      label?: string;
      helperText?: string;
      defaultValue?: unknown;
      autoFocus?: boolean;
    }
  >
>;

export type OpenAPIObjectProperties = {
  label?: string;
};

export type NormalizedPathMeta = {
  sourcePath?: string;
  targetPaths: string[];
};

export interface DataSource {
  provider: string;
}

export enum FieldType {
  Number = 'number',
  Select = 'select',
  Text = 'text',
  Hidden = 'hidden',
}

export enum GroupType {
  Accordion = 'accordion',
  Line = 'line',
}

interface CommonFieldParams {
  label?: string;
  defaultValue?: unknown;
  disabled?: boolean;
  autoFocus?: boolean;
  helperText?: string;
  // TODO support tooltip in the ui-schema for all components + documentation
  tooltip?: string;
  badge?: string;
  badgeToApi?: boolean;
  modes?: FieldParamsModeOverrides;
}

export interface NumberFieldParams extends CommonFieldParams {
  step?: number;
  placeholder?: string;
}

type SelectOptionsItem = { label: string; value: string };

type SelectFieldParamsBase = CommonFieldParams & {
  multiple?: boolean;
  displayEmpty?: boolean;
  defaultOpen?: boolean;
  readOnly?: boolean;
};

export type SelectFieldParams =
  | (SelectFieldParamsBase & {
      options?: SelectOptionsItem[];
      optionsPath?: never;
      optionsPathConfig?: never;
    })
  | (SelectFieldParamsBase & {
      optionsPath: string;
      optionsPathConfig: { labelPath: string; valuePath: string };
      options?: never;
    });

export interface TextFieldParams extends CommonFieldParams {
  placeholder?: string;
  multiline?: boolean;
  rows?: number;
  minRows?: number;
  maxRows?: number;
  type?: 'text' | 'password' | 'email' | 'search' | 'tel';
  readOnly?: boolean;
  variant?: 'outlined' | 'filled' | 'standard';
  // TODO size?: 'small' | 'medium';
  color?: 'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning';
  fullWidth?: boolean;
  hiddenLabel?: boolean;
  margin?: 'none' | 'dense' | 'normal';
}

export type FieldParamsMap = {
  [FieldType.Number]: NumberFieldParams;
  [FieldType.Select]: SelectFieldParams;
  [FieldType.Text]: TextFieldParams;
  [FieldType.Hidden]: CommonFieldParams;
};

type PathOrId =
  | { path: string | string[]; id?: never }
  | { id: string; path?: never };

export type CelExpression = {
  celExpr: string;
  message?: string;
};

export type RegexValidation = {
  pattern: string;
  message?: string;
};

export type CommonValidation = {
  required?: boolean;
  regex?: RegexValidation;
  celExpressions?: CelExpression[];
};

export type TextValidation = CommonValidation & {
  min?: number;
  max?: number;
  length?: number;
  email?: boolean;
  url?: boolean;
  uuid?: boolean;
  trim?: boolean;
  toLowerCase?: boolean;
  toUpperCase?: boolean;
};

export type NumberValidation = CommonValidation & {
  min?: number;
  max?: number;
  gt?: number;
  lt?: number;
  int?: boolean;
  multipleOf?: number;
  safe?: boolean;
};

export type ValidationMap = {
  [FieldType.Number]: NumberValidation;
  [FieldType.Text]: TextValidation;
  [FieldType.Select]: CommonValidation;
  [FieldType.Hidden]: CommonValidation;
};

export type ModeAwareValidation<T extends CommonValidation> = T & {
  modes?: Partial<Record<FormMode, T & { inheritShared?: boolean }>>;
};

export type Component = {
  [K in keyof FieldParamsMap]: {
    uiType: K;
    techPreview?: boolean;
    validation?: ModeAwareValidation<ValidationMap[K]>;
    fieldParams: FieldParamsMap[K];
    modes?: ComponentModeOverrides;
    _normalized?: NormalizedPathMeta;
    dataSource?: DataSource;
  } & PathOrId;
}[keyof FieldParamsMap];

export type ComponentGroup = {
  uiType: 'group' | 'hidden';
  label?: string;
  description?: string;
  groupType?: GroupType;
  //TODO check groupParams is work
  groupParams?: Record<string, unknown>;
  components: { [key: string]: Component | ComponentGroup };
  componentsOrder?: string[];
};

export type Section = {
  label?: string;
  description?: string;
  components: { [key: string]: Component | ComponentGroup };
  componentsOrder?: string[];
};

export type Topology = {
  sections: {
    [key: string]: Section;
  };
  sectionsOrder?: string[];
};

export type TopologyUISchemas = {
  // TODISCUSS
  // we can put Sections on the same level as topology key, but lefted for now, for case
  // if we will want more properties for topology
  [K in string]: Topology;
} & Record<string, unknown>;

export type UIGeneratorProps = {
  sectionKey: string;
  sections: { [key: string]: Section };
  providerObject?: Provider;
  loadingDefaultsForEdition?: boolean;
  formMode?: FormMode;
  namespace?: string;
};
