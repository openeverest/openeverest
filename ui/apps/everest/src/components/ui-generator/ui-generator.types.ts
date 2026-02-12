export type OpenAPIObjectProperties = {
  label?: string;
};

export enum FieldType {
  Number = 'number',
  Select = 'select',
  Hidden = 'hidden',
}

export enum GroupType {
  Accordion = 'accordion',
  Line = 'line',
}

interface CommonFieldParams {
  label?: string;
  defaultValue?: unknown;
  required?: boolean;
  disabled?: boolean;
  autoFocus?: boolean;
  helperText?: string;
}

export interface NumberFieldParams extends CommonFieldParams {
  step?: number;
  placeholder?: string;
  // badge?: string; https://github.com/openeverest/openeverest/issues/1854
}

interface SelectFieldParams extends CommonFieldParams {
  options: { label: string; value: string }[];
  multiple?: boolean; // Allow multi-select
}

export type FieldParamsMap = {
  [FieldType.Number]: NumberFieldParams;
  [FieldType.Select]: SelectFieldParams;
  [FieldType.Hidden]: CommonFieldParams;
};

type PathOrId = { path: string; id?: never } | { id: string; path?: never };

export type CelExpression = {
  celExpr: string;
  message?: string;
};

export type RegexValidation = {
  pattern: string;
  message?: string;
};

export type ValidationMap = {
  [FieldType.Number]: {
    min?: number;
    max?: number;
    gt?: number;
    lt?: number;
    int?: boolean;
    multipleOf?: number;
    safe?: boolean;
    regex?: RegexValidation;
    celExpressions?: CelExpression[];
  };
  [FieldType.Select]: {
    regex?: RegexValidation;
    celExpressions?: CelExpression[];
  };
  [FieldType.Hidden]: {
    regex?: RegexValidation;
    celExpressions?: CelExpression[];
  };
};

export type Component = {
  [K in keyof FieldParamsMap]: {
    uiType: K;
    techPreview?: boolean;
    validation?: ValidationMap[K];
    fieldParams: FieldParamsMap[K];
  } & PathOrId;
}[keyof FieldParamsMap];

export type ComponentGroup = {
  uiType: 'group' | 'hidden';
  label?: string;
  description?: string;
  groupType?: GroupType;
  //TODO check groupParams is work
  groupParams?: any;
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
} & Record<string, any>;
