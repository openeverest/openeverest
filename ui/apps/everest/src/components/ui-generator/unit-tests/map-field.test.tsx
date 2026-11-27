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

import { ReactNode } from 'react';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import { useCelValidation } from '../hooks/use-cel-validation';
import {
  ComponentGroup,
  FieldType,
  GroupType,
  TopologyUISchemas,
} from '../ui-generator.types';
import { keyValueRow } from 'components/ui-generator/test-fixtures/key-value-row.fixture';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';

const schema: TopologyUISchemas = {
  replicaSet: {
    sections: {
      advanced: {
        components: {
          annotations: {
            uiType: FieldType.Map,
            path: 'spec.proxy.annotations',
            fieldParams: {
              label: 'Annotations',
              addButtonLabel: 'Add annotation',
              defaultValue: { team: 'db' },
            },
            itemTemplate: keyValueRow(),
          },
        },
      },
    },
  },
};

// A map with rules on the count, so the map itself gets an error.
const validatedSchema: TopologyUISchemas = {
  replicaSet: {
    sections: {
      advanced: {
        components: {
          annotations: {
            uiType: FieldType.Map,
            path: 'spec.proxy.annotations',
            fieldParams: {
              label: 'Annotations',
              helperText: 'Added to the Service',
            },
            validation: { required: true, max: 1 },
            itemTemplate: keyValueRow(),
          },
        },
      },
    },
  },
};

const FormWrapper = ({
  children,
  uiSchema = schema,
  watchCelDependencies = true,
}: {
  children: ReactNode;
  uiSchema?: TopologyUISchemas;
  // The create wizard doesn't watch them.
  watchCelDependencies?: boolean;
}) => {
  const { schema: zodSchema, celDependencyGroups } = buildZodSchema(
    uiSchema,
    'replicaSet'
  );
  const methods = useForm<Record<string, unknown>>({
    resolver: zodResolver(zodSchema),
    mode: 'onChange',
    defaultValues: getDefaultValues(uiSchema, 'replicaSet'),
  });
  // As the forms built on the generator do it.
  useCelValidation(
    watchCelDependencies ? celDependencyGroups : [],
    methods.control,
    methods.trigger
  );
  return <FormProvider {...methods}>{children}</FormProvider>;
};

const type = async (name: string, value: string) =>
  act(async () => {
    fireEvent.change(screen.getByRole('textbox', { name }), {
      target: { value },
    });
  });

describe('UIGenerator - map field', () => {
  it(
    'flags a key set twice until either row changes it',
    { timeout: 15000 },
    async () => {
      render(
        <TestWrapper>
          <FormWrapper>
            <UIGenerator
              sections={schema.replicaSet.sections}
              sectionKey="advanced"
            />
          </FormWrapper>
        </TestWrapper>
      );

      expect(screen.getByRole('textbox', { name: 'Key 1' })).toHaveValue(
        'team'
      );

      fireEvent.click(screen.getByRole('button', { name: 'Add annotation' }));
      expect(screen.getByRole('textbox', { name: 'Key 2' })).toHaveFocus();
      await type('Key 2', 'team');

      expect(
        await screen.findByText('team is set more than once')
      ).toBeInTheDocument();

      // The error sits on row 2, so clearing it needs both rows re-validated.
      await type('Key 1', 'tier');

      expect(
        screen.queryByText('team is set more than once')
      ).not.toBeInTheDocument();
    }
  );

  it('leaves the focus alone when a row is removed after adding one', async () => {
    render(
      <TestWrapper>
        <FormWrapper>
          <UIGenerator
            sections={schema.replicaSet.sections}
            sectionKey="advanced"
          />
        </FormWrapper>
      </TestWrapper>
    );

    fireEvent.click(screen.getByRole('button', { name: 'Add annotation' }));
    expect(screen.getByRole('textbox', { name: 'Key 2' })).toHaveFocus();

    // The added row moves up to row 1 and remounts there.
    fireEvent.click(screen.getByRole('button', { name: 'Remove 1' }));
    expect(
      await screen.findByRole('textbox', { name: 'Key 1' })
    ).not.toHaveFocus();
  });

  it("shows the map's own error in place of the helper text", async () => {
    render(
      <TestWrapper>
        <FormWrapper uiSchema={validatedSchema}>
          <UIGenerator
            sections={validatedSchema.replicaSet.sections}
            sectionKey="advanced"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByText('Added to the Service')).toBeInTheDocument();

    // An added row left empty counts as no entry, so required fires.
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));
    expect(
      await screen.findByText('Add at least one entry')
    ).toBeInTheDocument();
    expect(screen.queryByText('Added to the Service')).not.toBeInTheDocument();

    await type('Key 1', 'team');
    expect(await screen.findByText('Added to the Service')).toBeInTheDocument();
    expect(
      screen.queryByText('Add at least one entry')
    ).not.toBeInTheDocument();

    // A count error and a row error show together: fixing the duplicate
    // doesn't need the count fixed first.
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));
    await type('Key 2', 'team');

    expect(
      await screen.findByText('Remove 1 entry (at most 1 allowed)')
    ).toBeInTheDocument();
    expect(screen.getByText('team is set more than once')).toBeInTheDocument();
  });

  it('flags a value without a key as soon as the value is typed', async () => {
    render(
      <TestWrapper>
        <FormWrapper>
          <UIGenerator
            sections={schema.replicaSet.sections}
            sectionKey="advanced"
          />
        </FormWrapper>
      </TestWrapper>
    );

    fireEvent.click(screen.getByRole('button', { name: 'Add annotation' }));
    // Let the validation the new row started settle, as it would before a
    // user types.
    await act(() => new Promise((resolve) => setTimeout(resolve, 0)));
    await type('Value 2', 'orphan');

    expect(await screen.findByText('Enter a key')).toBeInTheDocument();
  });

  it('renders a row as text fields that take any text param', () => {
    // The row's fields are the schema's own; `path` says which is the key.
    const envSchema: TopologyUISchemas = {
      replicaSet: {
        sections: {
          advanced: {
            components: {
              env: {
                uiType: FieldType.Map,
                path: 'spec.env',
                fieldParams: { label: 'Env', defaultValue: { TOKEN: 's3' } },
                itemTemplate: {
                  uiType: 'group',
                  groupType: GroupType.Line,
                  components: {
                    secret: {
                      uiType: FieldType.Text,
                      path: 'value',
                      fieldParams: { type: 'password' },
                    },
                    variable: {
                      uiType: FieldType.Text,
                      path: 'key',
                      fieldParams: { placeholder: 'NAME' },
                    },
                  },
                },
              },
            },
          },
        },
      },
    };

    render(
      <TestWrapper>
        <FormWrapper uiSchema={envSchema}>
          <UIGenerator
            sections={envSchema.replicaSet.sections}
            sectionKey="advanced"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByLabelText('Value 1')).toHaveAttribute(
      'type',
      'password'
    );
    expect(screen.getByRole('textbox', { name: 'Key 1' })).toHaveAttribute(
      'placeholder',
      'NAME'
    );
  });

  it('separates stacked rows with a line, and rows in a line with none', () => {
    const withRows = (itemTemplate: ComponentGroup): TopologyUISchemas => ({
      replicaSet: {
        sections: {
          advanced: {
            components: {
              annotations: {
                uiType: FieldType.Map,
                path: 'spec.proxy.annotations',
                fieldParams: {
                  label: 'Annotations',
                  defaultValue: { team: 'db', tier: 'prod' },
                },
                itemTemplate,
              },
            },
          },
        },
      },
    });
    const renderMap = (uiSchema: TopologyUISchemas) =>
      render(
        <TestWrapper>
          <FormWrapper uiSchema={uiSchema}>
            <UIGenerator
              sections={uiSchema.replicaSet.sections}
              sectionKey="advanced"
            />
          </FormWrapper>
        </TestWrapper>
      );

    // No groupType: the row's inputs are stacked.
    const { unmount } = renderMap(
      withRows({ ...keyValueRow(), groupType: undefined })
    );
    expect(screen.getAllByRole('separator')).toHaveLength(1);
    unmount();

    renderMap(withRows(keyValueRow()));
    expect(screen.queryByRole('separator')).not.toBeInTheDocument();
  });

  it('disables the rows and buttons of a disabled map', () => {
    const disabledSchema: TopologyUISchemas = {
      replicaSet: {
        sections: {
          advanced: {
            components: {
              annotations: {
                uiType: FieldType.Map,
                path: 'spec.proxy.annotations',
                fieldParams: {
                  label: 'Annotations',
                  defaultValue: { team: 'db' },
                  disabled: true,
                },
                itemTemplate: keyValueRow(),
              },
            },
          },
        },
      },
    };

    render(
      <TestWrapper>
        <FormWrapper uiSchema={disabledSchema}>
          <UIGenerator
            sections={disabledSchema.replicaSet.sections}
            sectionKey="advanced"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByRole('textbox', { name: 'Key 1' })).toBeDisabled();
    expect(screen.getByRole('textbox', { name: 'Value 1' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Remove 1' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Add' })).toBeDisabled();
  });

  it('runs the rule of a field that reads the map again as its rows change', async () => {
    const ruledSchema: TopologyUISchemas = {
      replicaSet: {
        sections: {
          advanced: {
            components: {
              ...schema.replicaSet.sections.advanced.components,
              owner: {
                uiType: FieldType.Text,
                path: 'spec.proxy.owner',
                fieldParams: { label: 'Owner' },
                validation: {
                  celExpressions: [
                    {
                      // Names a key inside the map, where the form has no field.
                      celExpr: 'has(spec.proxy.annotations.team)',
                      message: 'Add a team annotation',
                    },
                  ],
                },
              },
            },
          },
        },
      },
    };

    render(
      <TestWrapper>
        <FormWrapper uiSchema={ruledSchema}>
          <UIGenerator
            sections={ruledSchema.replicaSet.sections}
            sectionKey="advanced"
          />
        </FormWrapper>
      </TestWrapper>
    );

    await type('Key 1', 'tier');

    expect(
      await screen.findByText('Add a team annotation')
    ).toBeInTheDocument();

    await type('Key 1', 'team');

    await waitFor(() =>
      expect(
        screen.queryByText('Add a team annotation')
      ).not.toBeInTheDocument()
    );
  });

  it("shows the map's own rule under its rows, and checks it again as a value changes", async () => {
    const ruledSchema: TopologyUISchemas = {
      replicaSet: {
        sections: {
          advanced: {
            components: {
              annotations: {
                uiType: FieldType.Map,
                path: 'spec.proxy.annotations',
                fieldParams: {
                  label: 'Annotations',
                  helperText: 'Added to the Service',
                  defaultValue: { team: 'db' },
                },
                validation: {
                  celExpressions: [
                    {
                      celExpr:
                        "!has(spec.proxy.annotations) || !('team' in spec.proxy.annotations) || spec.proxy.annotations['team'] != ''",
                      message: 'Name the team',
                    },
                  ],
                },
                itemTemplate: keyValueRow(),
              },
            },
          },
        },
      },
    };

    render(
      <TestWrapper>
        {/* The map checks its own rule, with nothing watching for it. */}
        <FormWrapper uiSchema={ruledSchema} watchCelDependencies={false}>
          <UIGenerator
            sections={ruledSchema.replicaSet.sections}
            sectionKey="advanced"
          />
        </FormWrapper>
      </TestWrapper>
    );

    await type('Value 1', '');

    expect(await screen.findByText('Name the team')).toBeInTheDocument();
    expect(screen.queryByText('Added to the Service')).not.toBeInTheDocument();

    await type('Value 1', 'db');

    await waitFor(() =>
      expect(screen.queryByText('Name the team')).not.toBeInTheDocument()
    );
    expect(screen.getByText('Added to the Service')).toBeInTheDocument();
  });
});
