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

import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { TestWrapper } from 'utils/test';
import { Button } from '@mui/material';
import { getDBWizardSchema } from '../database-form-schema';
import { buildZodSchema } from 'components/ui-generator/utils/schema-builder';
import {
  FieldType,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { DbWizardFormFields } from 'consts';
import { getDefaultValues } from 'components/ui-generator/utils/default-values';

vi.mock('components/ui-generator/utils/schema-builder/cel-validation', () => ({
  extractCelFieldPaths: vi.fn(() => []),
  validateCelExpression: vi.fn(() => ({ isValid: true })),
}));

const BASE_DEFAULTS = {
  [DbWizardFormFields.provider]: 'psmdb',
  [DbWizardFormFields.dbName]: 'my-db',
  [DbWizardFormFields.k8sNamespace]: 'default',
  topology: { type: 'replica' },
};

const makeUiSchema = (
  fieldParams: Record<string, unknown> = {},
  validation: Record<string, unknown> = {}
): TopologyUISchemas => ({
  replica: {
    sections: {
      resources: {
        label: 'Resources',
        components: {
          nodes: {
            uiType: FieldType.Number,
            path: 'resources.nodes',
            fieldParams: { label: 'Nodes', ...fieldParams },
            validation: Object.keys(validation).length ? validation : undefined,
          },
        },
      },
    },
    sectionsOrder: ['resources'],
  },
});

interface WrapperProps {
  children: React.ReactNode;
  uiSchema?: TopologyUISchemas;
  extraDefaults?: Record<string, unknown>;
  onSubmit?: (data: Record<string, unknown>) => void;
}

const Wrapper = ({
  children,
  uiSchema,
  extraDefaults = {},
  onSubmit = vi.fn(),
}: WrapperProps) => {
  const openApiSchema = uiSchema
    ? buildZodSchema(uiSchema, 'replica').schema
    : undefined;

  const combinedSchema = getDBWizardSchema([], false, openApiSchema);

  const dynamicDefaults = uiSchema ? getDefaultValues(uiSchema, 'replica') : {};
  const defaultValues = {
    ...BASE_DEFAULTS,
    ...dynamicDefaults,
    ...extraDefaults,
  };

  const methods = useForm({
    mode: 'onChange',
    reValidateMode: 'onChange',
    resolver: zodResolver(combinedSchema),
    defaultValues,
  });

  return (
    <TestWrapper>
      <FormProvider {...methods}>
        <form onSubmit={methods.handleSubmit(onSubmit)}>
          {children}
          <Button
            type="submit"
            disabled={!methods.formState.isValid}
            data-testid="submit-button"
          >
            Submit
          </Button>
        </form>
      </FormProvider>
    </TestWrapper>
  );
};

describe('getDBWizardSchema + openApiValidationSchema combination', () => {
  it('is valid when base fields are filled and no openApi schema is provided', () => {
    const schema = getDBWizardSchema([], false);
    const result = schema.safeParse(BASE_DEFAULTS);
    expect(result.success).toBe(true);
  });

  it('is valid when a required number field has a numeric value', () => {
    const uiSchema = makeUiSchema({}, { required: true });
    const openApiSchema = buildZodSchema(uiSchema, 'replica').schema;
    const schema = getDBWizardSchema([], false, openApiSchema);

    const result = schema.safeParse({
      ...BASE_DEFAULTS,
      resources: { nodes: 3 },
    });
    expect(result.success).toBe(true);
  });

  it('is valid when a required number field has a string-number value (as provided by inputs)', () => {
    const uiSchema = makeUiSchema({}, { required: true });
    const openApiSchema = buildZodSchema(uiSchema, 'replica').schema;
    const schema = getDBWizardSchema([], false, openApiSchema);

    // String "5" from a text input must coerce and succeed — the old .and() approach failed here
    const result = schema.safeParse({
      ...BASE_DEFAULTS,
      resources: { nodes: '5' },
    });
    expect(result.success).toBe(true);
  });

  it('does NOT produce an invalid_intersection_types error at root path', () => {
    const uiSchema = makeUiSchema({}, { required: true });
    const openApiSchema = buildZodSchema(uiSchema, 'replica').schema;
    const schema = getDBWizardSchema([], false, openApiSchema);

    const result = schema.safeParse({
      ...BASE_DEFAULTS,
      resources: { nodes: '5' },
    });

    if (!result.success) {
      const rootIntersectionIssue = result.error.issues.find(
        (i) => i.path.length === 0 && i.code === 'invalid_intersection_types'
      );
      expect(rootIntersectionIssue).toBeUndefined();
    }
    expect(result.success).toBe(true);
  });

  it('is invalid when a required number field is empty string', () => {
    const uiSchema = makeUiSchema({}, { required: true });
    const openApiSchema = buildZodSchema(uiSchema, 'replica').schema;
    const schema = getDBWizardSchema([], false, openApiSchema);

    const result = schema.safeParse({
      ...BASE_DEFAULTS,
      resources: { nodes: '' },
    });
    expect(result.success).toBe(false);
  });

  it('reports errors at the field path (not root) for an invalid required number field', () => {
    const uiSchema = makeUiSchema({}, { required: true });
    const openApiSchema = buildZodSchema(uiSchema, 'replica').schema;
    const schema = getDBWizardSchema([], false, openApiSchema);

    const result = schema.safeParse({
      ...BASE_DEFAULTS,
      resources: { nodes: '' },
    });
    expect(result.success).toBe(false);

    if (!result.success) {
      const rootIssue = result.error.issues.find((i) => i.path.length === 0);
      expect(rootIssue).toBeUndefined();

      const fieldIssue = result.error.issues.find((i) => i.path.length > 0);
      expect(fieldIssue).toBeDefined();
    }
  });

  it('is valid when an optional number field is undefined (explicitly absent)', () => {
    const uiSchema = makeUiSchema({}, { required: false });
    const openApiSchema = buildZodSchema(uiSchema, 'replica').schema;
    const schema = getDBWizardSchema([], false, openApiSchema);

    // undefined is the canonical "absent" value for optional fields
    const result = schema.safeParse({
      ...BASE_DEFAULTS,
      resources: { nodes: undefined },
    });
    expect(result.success).toBe(true);
  });
});

describe('Database form validation', () => {
  it('submit is enabled when all base fields are valid and no dynamic schema', async () => {
    render(
      <Wrapper>
        <span />
      </Wrapper>
    );

    await waitFor(() => {
      expect(screen.getByTestId('submit-button')).not.toBeDisabled();
    });
  });

  it('required number field: click submit when empty → submit stays disabled', async () => {
    // Tests that a required empty number field blocks submission
    const uiSchema = makeUiSchema({}, { required: true });
    const mockSubmit = vi.fn();

    render(
      <Wrapper uiSchema={uiSchema} onSubmit={mockSubmit}>
        <UIGenerator
          sections={uiSchema.replica!.sections}
          sectionKey="resources"
        />
      </Wrapper>
    );

    const submitButton = screen.getByTestId('submit-button');

    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockSubmit).not.toHaveBeenCalled();
      expect(submitButton).toBeDisabled();
    });
  });

  it('required number field: clear then refill → submit re-enables and can be submitted', async () => {
    // This is the core bug regression test: previously, after clearing + refilling
    // a number field, the form would get a root-level "invalid_intersection_types"
    // error making it impossible to submit even though it appeared valid.
    const uiSchema = makeUiSchema({}, { required: true });
    const mockSubmit = vi.fn();

    render(
      <Wrapper
        uiSchema={uiSchema}
        extraDefaults={{ resources: { nodes: 3 } }}
        onSubmit={mockSubmit}
      >
        <UIGenerator
          sections={uiSchema.replica!.sections}
          sectionKey="resources"
        />
      </Wrapper>
    );

    const input = screen.getByLabelText('Nodes');
    const submitButton = screen.getByTestId('submit-button');

    // Trigger validation by typing a valid value → form becomes valid
    fireEvent.change(input, { target: { value: '3' } });
    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });

    // Clear the field → form becomes invalid
    fireEvent.change(input, { target: { value: '' } });
    await waitFor(() => {
      expect(submitButton).toBeDisabled();
    });

    // Refill the field → form must become valid again (no invalid_intersection_types)
    fireEvent.change(input, { target: { value: '5' } });
    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });

    // Submit must succeed
    fireEvent.click(submitButton);
    await waitFor(() => {
      expect(mockSubmit).toHaveBeenCalledTimes(1);
    });
  });

  it('optional number field: fill value then submit succeeds', async () => {
    const uiSchema = makeUiSchema({}, { required: false });
    const mockSubmit = vi.fn();

    render(
      <Wrapper uiSchema={uiSchema} onSubmit={mockSubmit}>
        <UIGenerator
          sections={uiSchema.replica!.sections}
          sectionKey="resources"
        />
      </Wrapper>
    );

    const input = screen.getByLabelText('Nodes');
    const submitButton = screen.getByTestId('submit-button');

    // Enter a valid value to trigger validation and make the form valid
    fireEvent.change(input, { target: { value: '10' } });
    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });

    fireEvent.click(submitButton);
    await waitFor(() => {
      expect(mockSubmit).toHaveBeenCalled();
    });
  });
});

// ── topology-switch error path regression tests ───────────────────────────────
//
// BUG: When the user switches to a topology whose nested form structure has not
// yet been seeded into the form (e.g. spec.components.configServer is undefined),
// Zod reports `invalid_type` at the PARENT path (spec.components.configServer)
// instead of at the LEAF path (spec.components.configServer.replicas). This
// causes the wrong step to show an error icon.
//
// FIX:
//   1. buildSectionFieldMap now registers all intermediate path prefixes, so
//      errors at any level of a path hierarchy resolve to the correct step.
//   2. DatabasePage resets the form with topology defaults on topology change,
//      ensuring every nested object is properly initialised before validation.

const makeShardedUiSchema = (): TopologyUISchemas => ({
  sharded: {
    sections: {
      components: {
        label: 'Components',
        components: {
          configServerReplicas: {
            uiType: FieldType.Number,
            path: 'spec.components.configServer.replicas',
            fieldParams: { label: 'Config Server Replicas' },
            validation: { required: true },
          },
          proxyReplicas: {
            uiType: FieldType.Number,
            path: 'spec.components.proxy.replicas',
            fieldParams: { label: 'Proxy Replicas' },
            validation: { required: true },
          },
        },
      },
    },
    sectionsOrder: ['components'],
  },
});

describe('topology-switch nested object error path', () => {
  it('when a nested container object is UNDEFINED Zod reports error at the container path, not the leaf', () => {
    // This documents the root cause: if spec.components.configServer is
    // undefined in form data, Zod cannot reach the leaf field replicas
    // and instead reports invalid_type at the container level.
    const uiSchema = makeShardedUiSchema();
    const openApiSchema = buildZodSchema(uiSchema, 'sharded').schema;
    const schema = getDBWizardSchema([], false, openApiSchema);

    const result = schema.safeParse({
      ...BASE_DEFAULTS,
      // spec.components.configServer is intentionally absent (undefined)
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      // The error is reported at spec.components.configServer, NOT at
      // spec.components.configServer.replicas, because the parent is absent.
      const containerIssue = result.error.issues.find(
        (i) =>
          i.path.join('.') === 'spec.components.configServer' ||
          i.path.join('.') === 'spec.components' ||
          i.path.join('.') === 'spec'
      );
      expect(containerIssue).toBeDefined();
    }
  });

  it('when nested objects ARE initialized, errors appear at the correct leaf path', () => {
    // After topology-switch seeding (mergeTopologyDefaults), the nested objects
    // exist as {}, so Zod can validate the leaf field and report the error there.
    const uiSchema = makeShardedUiSchema();
    const openApiSchema = buildZodSchema(uiSchema, 'sharded').schema;
    const schema = getDBWizardSchema([], false, openApiSchema);

    const result = schema.safeParse({
      ...BASE_DEFAULTS,
      spec: {
        components: {
          // containers exist but replicas is empty → error should be at leaf
          configServer: { replicas: '' },
          proxy: { replicas: '' },
        },
      },
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      // Errors must be at the leaf paths, not at intermediate ones
      const paths = result.error.issues.map((i) => i.path.join('.'));
      // No error should be reported at an intermediate-only path
      expect(paths.some((p) => p === 'spec')).toBe(false);
      expect(paths.some((p) => p === 'spec.components')).toBe(false);
      expect(paths.some((p) => p === 'spec.components.configServer')).toBe(
        false
      );

      // Errors are at the actual field paths
      expect(
        paths.some((p) => p === 'spec.components.configServer.replicas')
      ).toBe(true);
    }
  });

  it('getDefaultValues produces leaf-level defaults that initialize nested objects', () => {
    // This verifies that getDefaultValues (used in mergeTopologyDefaults) correctly
    // seeds all nested objects, preventing the "container is undefined" Zod error.
    const uiSchema = makeShardedUiSchema();
    const defaults = getDefaultValues(uiSchema, 'sharded');

    // Defaults should produce the nested structure, not flat keys
    expect(defaults).toHaveProperty('spec');
    const spec = defaults.spec as Record<string, unknown>;
    expect(spec).toHaveProperty('components');
    const components = spec.components as Record<string, unknown>;
    expect(components).toHaveProperty('configServer');
    expect(components).toHaveProperty('proxy');
  });
});
