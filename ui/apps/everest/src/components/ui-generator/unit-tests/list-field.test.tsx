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
import { parse } from 'yaml';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import { useCelValidation } from '../hooks/use-cel-validation';
import {
  FieldType,
  FormMode,
  ListFieldParams,
  ListValidation,
  TextFieldParams,
  TopologyUISchemas,
} from '../ui-generator.types';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { preprocessSchema } from '../utils/preprocess/preprocess-schema';

const CIDR = { pattern: '^[0-9./]+$', message: 'Use CIDR notation' };

// A list of source ranges that opens with one entry, as the form of
// `formMode` gets it: with `fieldParams.modes` applied to the list and to its
// item.
const rangesSchema = (
  {
    list,
    item,
    validation,
  }: {
    list?: Partial<ListFieldParams>;
    item?: TextFieldParams;
    validation?: ListValidation;
  } = {},
  formMode: FormMode = FormMode.New
): TopologyUISchemas =>
  preprocessSchema(
    {
      replicaSet: {
        sections: {
          advanced: {
            components: {
              sourceRanges: {
                uiType: FieldType.List,
                path: 'spec.proxy.sourceRanges',
                fieldParams: {
                  label: 'Source ranges',
                  addButtonLabel: 'Add range',
                  helperText: 'Who may connect',
                  defaultValue: ['10.0.0.0/8'],
                  ...list,
                },
                validation,
                itemTemplate: {
                  uiType: FieldType.Text,
                  fieldParams: item,
                  validation: { regex: CIDR },
                },
              },
            },
          },
        },
      },
    },
    undefined,
    formMode
  );

const FormWrapper = ({
  children,
  uiSchema,
}: {
  children: ReactNode;
  uiSchema: TopologyUISchemas;
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
  useCelValidation(celDependencyGroups, methods.control, methods.trigger);
  return <FormProvider {...methods}>{children}</FormProvider>;
};

const listForm = (uiSchema: TopologyUISchemas) => (
  <TestWrapper>
    <FormWrapper uiSchema={uiSchema}>
      <UIGenerator
        sections={uiSchema.replicaSet.sections}
        sectionKey="advanced"
      />
    </FormWrapper>
  </TestWrapper>
);

const row = (number: number) =>
  screen.getByRole('textbox', { name: `Value ${number}` });

const type = async (number: number, value: string) =>
  act(async () => {
    fireEvent.change(row(number), { target: { value } });
  });

const add = () =>
  fireEvent.click(screen.getByRole('button', { name: 'Add range' }));

// Lets the validation a new row started settle, as it would before a user
// types.
const settle = () =>
  act(() => new Promise((resolve) => setTimeout(resolve, 0)));

describe('UIGenerator - list field', () => {
  it('opens with the default entries, and adds, fills and removes rows', async () => {
    render(listForm(rangesSchema()));

    expect(row(1)).toHaveValue('10.0.0.0/8');

    add();
    expect(row(2)).toHaveFocus();
    await type(2, '192.168.0.0/16');

    // The second row moves up, with what it holds.
    fireEvent.click(screen.getByRole('button', { name: 'Remove 1' }));
    await waitFor(() =>
      expect(
        screen.queryByRole('textbox', { name: 'Value 2' })
      ).not.toBeInTheDocument()
    );
    expect(row(1)).toHaveValue('192.168.0.0/16');
    expect(row(1)).not.toHaveFocus();
  });

  it("shows the item's error under its row, and the list's own in place of the helper text", async () => {
    render(listForm(rangesSchema({ validation: { max: 1 } })));

    expect(screen.getByText('Who may connect')).toBeInTheDocument();

    await type(1, 'office');
    expect(await screen.findByText(CIDR.message)).toBeInTheDocument();
    expect(screen.getByText('Who may connect')).toBeInTheDocument();

    // A count error and a row error show together.
    add();
    await settle();
    await type(2, '192.168.0.0/16');
    expect(
      await screen.findByText('Remove 1 entry (at most 1 allowed)')
    ).toBeInTheDocument();
    expect(screen.queryByText('Who may connect')).not.toBeInTheDocument();
    expect(screen.getByText(CIDR.message)).toBeInTheDocument();
  });

  it(
    'flags a value added twice in a unique list until either row changes it',
    { timeout: 15000 },
    async () => {
      render(listForm(rangesSchema({ validation: { unique: true } })));

      add();
      await settle();
      await type(2, '10.0.0.0/8');
      expect(
        await screen.findByText('10.0.0.0/8 is added more than once')
      ).toBeInTheDocument();

      // The error sits on row 2, so clearing it needs both rows re-validated.
      await type(1, '10.0.0.0/16');
      await waitFor(() =>
        expect(
          screen.queryByText('10.0.0.0/8 is added more than once')
        ).not.toBeInTheDocument()
      );
    }
  );

  it('lets a value repeat in a list that is not unique', async () => {
    render(listForm(rangesSchema({ validation: { max: 1 } })));

    add();
    await settle();
    await type(2, '10.0.0.0/8');

    // The list is validated: the count error shows, and no duplicate with it.
    expect(
      await screen.findByText('Remove 1 entry (at most 1 allowed)')
    ).toBeInTheDocument();
    expect(screen.queryByText(/more than once/)).not.toBeInTheDocument();
  });

  it("checks the list's own rule again as a row changes, with nothing watching for it", async () => {
    const uiSchema: TopologyUISchemas = preprocessSchema(
      parse(`
        replicaSet:
          sections:
            advanced:
              components:
                sourceRanges:
                  uiType: list
                  path: spec.proxy.sourceRanges
                  fieldParams:
                    label: Source ranges
                    helperText: Who may connect
                    defaultValue: [10.0.0.0/8]
                  validation:
                    celExpressions:
                      - celExpr: >-
                          !has(spec.proxy.sourceRanges) ||
                          !('0.0.0.0/0' in spec.proxy.sourceRanges)
                        message: Don't open the service to everyone
                  itemTemplate: { uiType: text }
      `)
    );
    // The create wizard doesn't watch a rule's dependencies.
    const Form = ({ children }: { children: ReactNode }) => {
      const methods = useForm<Record<string, unknown>>({
        resolver: zodResolver(buildZodSchema(uiSchema, 'replicaSet').schema),
        mode: 'onChange',
        defaultValues: getDefaultValues(uiSchema, 'replicaSet'),
      });
      return <FormProvider {...methods}>{children}</FormProvider>;
    };
    render(
      <TestWrapper>
        <Form>
          <UIGenerator
            sections={uiSchema.replicaSet.sections}
            sectionKey="advanced"
          />
        </Form>
      </TestWrapper>
    );

    await type(1, '0.0.0.0/0');
    expect(
      await screen.findByText("Don't open the service to everyone")
    ).toBeInTheDocument();
    expect(screen.queryByText('Who may connect')).not.toBeInTheDocument();

    await type(1, '10.0.0.0/8');
    await waitFor(() =>
      expect(
        screen.queryByText("Don't open the service to everyone")
      ).not.toBeInTheDocument()
    );
    expect(screen.getByText('Who may connect')).toBeInTheDocument();
  });

  it('draws the item as the text field it declares', () => {
    render(
      listForm(
        rangesSchema({ item: { type: 'password', placeholder: 'Token' } })
      )
    );

    const input = screen.getByLabelText('Value 1');
    expect(input).toHaveAttribute('type', 'password');
    expect(input).toHaveAttribute('placeholder', 'Token');
  });

  it('renders nothing for a list this version does not support, and the fields next to it as usual', () => {
    // As forms that skip preprocess get it.
    const uiSchema: TopologyUISchemas = parse(`
      replicaSet:
        sections:
          advanced:
            components:
              users:
                uiType: list
                path: spec.users
                fieldParams: { label: Users, defaultValue: [{ name: admin }] }
                itemTemplate:
                  uiType: group
                  components:
                    name: { uiType: text, path: name }
              owner:
                uiType: text
                path: spec.owner
                fieldParams: { label: Owner }
    `);

    render(listForm(uiSchema));

    expect(screen.getByRole('textbox', { name: 'Owner' })).toBeInTheDocument();
    expect(screen.queryByText('Users')).not.toBeInTheDocument();
    expect(screen.getAllByRole('textbox')).toHaveLength(1);
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });
});

describe('UIGenerator - list field with locked rows', () => {
  const ITEM_FIXED_IN_EDIT: TextFieldParams = {
    modes: { edit: { disabled: true } },
  };

  it('fixes the entries the form opened with, and lets the user type in a row they add', async () => {
    render(listForm(rangesSchema({ item: ITEM_FIXED_IN_EDIT }, FormMode.Edit)));

    expect(row(1)).toBeDisabled();

    add();
    expect(row(2)).toBeEnabled();
    expect(row(2)).toHaveFocus();

    // The item's own rule checks the new row, though it skips the fixed ones.
    await settle();
    await type(2, 'office');
    expect(await screen.findByText(CIDR.message)).toBeInTheDocument();
  });

  it('leaves the item editable in a mode that does not fix it', () => {
    render(listForm(rangesSchema({ item: ITEM_FIXED_IN_EDIT }, FormMode.New)));

    expect(row(1)).toBeEnabled();
  });

  it('keeps Remove off for the rows the form opened with, and on for a row added since', async () => {
    render(listForm(rangesSchema({ list: { disableRemove: true } })));

    expect(screen.getByRole('button', { name: 'Remove 1' })).toBeDisabled();

    add();
    expect(screen.getByRole('button', { name: 'Remove 2' })).toBeEnabled();

    // Emptying the locked row would remove its entry all the same.
    await settle();
    await type(1, '');
    expect(await screen.findByText('Enter a value')).toBeInTheDocument();
  });

  it('disables Add alone, in the mode that says so', () => {
    const list: Partial<ListFieldParams> = {
      modes: { edit: { disableAdd: true } },
    };

    const { unmount } = render(listForm(rangesSchema({ list }, FormMode.Edit)));
    expect(screen.getByRole('button', { name: 'Add range' })).toBeDisabled();
    expect(row(1)).toBeEnabled();
    expect(screen.getByRole('button', { name: 'Remove 1' })).toBeEnabled();
    unmount();

    render(listForm(rangesSchema({ list }, FormMode.New)));
    expect(screen.getByRole('button', { name: 'Add range' })).toBeEnabled();
  });

  it('disables every input and button of a disabled list, a row the user added too', () => {
    const { rerender } = render(
      listForm(rangesSchema({ item: ITEM_FIXED_IN_EDIT }, FormMode.Edit))
    );
    add();
    expect(row(2)).toBeEnabled();

    // The same form, now drawing the list disabled.
    rerender(
      listForm(
        rangesSchema(
          { list: { disabled: true }, item: ITEM_FIXED_IN_EDIT },
          FormMode.Edit
        )
      )
    );

    expect(row(1)).toBeDisabled();
    expect(row(2)).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Remove 2' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Add range' })).toBeDisabled();
  });
});
