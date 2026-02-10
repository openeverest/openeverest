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

type BaseFieldParams = {
  label?: string;
  description?: string;
  defaultValue?: unknown;
};

type FieldParamsMap = {
  // TODO & TextFieldProps (https://mui.com/material-ui/api/text-field/)
  //  idially it should give access to all native mui properties that require a string value
  // (without references to root, react components, etc.)
  number: BaseFieldParams & {
    maxLength?: number;
    placeholder?: string;
    defaultValue?: number;
    badge?: string;
  };
  select: BaseFieldParams & {
    options: { label: string; value: string }[];
  };
  // TODO probably it's better to have others as a separate object to avoid duplication, but let's see
  // other: BaseFieldParams & {
  //     badge?: string;
  //     options?: { label: string; value: string }[];
  //     placeholder?: string;
  //     defaultValue?: string;
  // };
};

type PathOrId = { path: string; id?: never } | { id: string; path?: never };

export type CelExpression = {
  celExpr: string;
  message?: string;
};

export type Component = {
  [K in keyof FieldParamsMap]: {
    uiType: K;
    techPreview?: boolean;
    validation?: {
      min?: number;
      max?: number;
      celExpressions?: CelExpression[];
    };
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
