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

import { ComponentType, ReactNode } from 'react';
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
  Toggle = 'toggle',
  Hidden = 'hidden',
}

export enum WidgetType {
  // Public marker a provider authors to place the scheduling section; the
  // consumer expands it (per-component support, tabs) at render time.
  PodSchedulingPolicy = 'podSchedulingPolicy',
  // Internal to the scheduling orchestrator's own UIGenerator, not authored by
  // providers.
  Affinity = 'affinity',
}

// Host-rendered component; `widgetType` picks the renderer, like `groupType`
// picks a group's layout.
export const WIDGET_UI_TYPE = 'widget' as const;

export enum GroupType {
  Accordion = 'accordion',
  Bordered = 'bordered',
  Line = 'line',
  Toggleable = 'toggleable',
}

interface CommonFieldParams {
  label?: string;
  defaultValue?: unknown;
  disabled?: boolean;
  autoFocus?: boolean;
  helperText?: string;
  // TODO support tooltip in the ui-schema for all components + documentation
  tooltip?: string;
  /**
   * Informational text shown as an InfoOutlined icon next to the field.
   * Hovering over the icon reveals the text in a tooltip.
   * Use this to provide additional context or guidance to the plugin developer
   * without cluttering the field label or helperText.
   */
  info?: string;
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

export type ToggleFieldParams = CommonFieldParams;

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

// Toggles are always optional booleans, so `required` is unsupported. Typing it
// as `never` forbids setting it (a compile error), while keeping the property
// present so the generic validation pipeline can still read
// `validation.required` uniformly across field types.
export type ToggleValidation = Omit<CommonValidation, 'required'> & {
  required?: never;
};

export type FieldParamsMap = {
  [FieldType.Number]: NumberFieldParams;
  [FieldType.Select]: SelectFieldParams;
  [FieldType.Text]: TextFieldParams;
  [FieldType.Toggle]: ToggleFieldParams;
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
  [FieldType.Toggle]: ToggleValidation;
  [FieldType.Hidden]: CommonValidation;
};

type ComponentCommonFields = {
  techPreview?: boolean;
  modes?: ComponentModeOverrides;
  _normalized?: NormalizedPathMeta;
  dataSource?: DataSource;
};

export type ModeAwareValidation<T extends CommonValidation> = T & {
  modes?: Partial<Record<FormMode, T & { inheritShared?: boolean }>>;
};

type FieldComponent = {
  [K in keyof FieldParamsMap]: ComponentCommonFields & {
    uiType: K;
    validation?: ModeAwareValidation<ValidationMap[K]>;
    fieldParams: FieldParamsMap[K];
  } & PathOrId;
}[keyof FieldParamsMap];

export interface WidgetTarget {
  // Display key, e.g. the Instance component the value belongs to.
  key: string;
  path: string;
}

export type WidgetComponent = ComponentCommonFields & {
  uiType: typeof WIDGET_UI_TYPE;
  widgetType: WidgetType;
  validation?: CommonValidation;
  // No widget type reads schema params yet; add per-widget params here when one does.
  fieldParams?: never;
  // API paths a marker widget writes, resolved by preprocess from the provider.
  _widgetTargets?: WidgetTarget[];
  // A pure marker widget (e.g. podSchedulingPolicy) binds no value.
} & (PathOrId | { path?: never; id?: never });

export type Component = FieldComponent | WidgetComponent;

export const isWidgetComponent = (item: Component): item is WidgetComponent =>
  item.uiType === WIDGET_UI_TYPE;

export type WidgetRendererProps = {
  // Engine-resolved RHF field key. A widget is a first-class component: it flows
  // through the same preprocess / name-resolution / render pipeline as a field,
  // and only its final render is delegated to a host renderer. Handing over
  // `name` keeps the widget on that shared pipeline, so it (or parts of it) can
  // later migrate to plain schema fields without reworking name/path handling.
  name: string;
  item: WidgetComponent;
};
export type WidgetRenderer = ComponentType<WidgetRendererProps>;
export type WidgetRegistry = Partial<Record<WidgetType, WidgetRenderer>>;

export type ComponentGroup = {
  uiType: 'group' | 'hidden';
  label?: string;
  description?: string;
  groupType?: GroupType;
  //TODO check groupParams is work
  groupParams?: Record<string, unknown>;
  components: { [key: string]: Component | ComponentGroup };
  componentsOrder?: string[];
  _toggleable?: ToggleableMeta;
};

export interface ToggleableMeta {
  // Form-only switch field, e.g. `toggleable-switches.advanced~monitoring`.
  switchName: string;
  // Every API path written by fields nested (at any depth) inside the group.
  childPaths: string[];
}

// What UIGroup forwards to every groupType wrapper.
export interface GroupWrapperProps {
  children: ReactNode;
  label?: string;
  description?: string;
  // Set only for an active toggleable group.
  toggleable?: ToggleableMeta;
}

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
  emptySectionMessage?: ReactNode;
  widgetRegistry?: WidgetRegistry;
};
