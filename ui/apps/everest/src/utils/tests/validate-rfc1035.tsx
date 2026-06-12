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

import React from 'react';
import { fireEvent, render, waitFor, screen } from '@testing-library/react';
import { TestWrapper } from '../test';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const FormProviderWrapper = ({
  children,
  schema,
}: {
  children: React.ReactNode;
  schema: z.ZodSchema;
}) => {
  const methods = useForm({
    mode: 'onChange',
    resolver: zodResolver(schema),
  });

  return (
    <FormProvider {...methods}>
      <form>{children}</form>
    </FormProvider>
  );
};

export const validateInputWithRFC1035 = ({
  renderComponent,
  suiteName,
  errors,
  schema,
}: {
  renderComponent: () => JSX.Element;
  suiteName: string;
  errors: Record<string, string>;
  schema?: z.ZodSchema;
}) => {
  describe(suiteName, () => {
    beforeEach(() => {
      const component = renderComponent();
      render(
        <TestWrapper>
          {schema ? (
            <FormProviderWrapper schema={schema}>
              {component}
            </FormProviderWrapper>
          ) : (
            component
          )}
        </TestWrapper>
      );
    });

    describe('name input', () => {
      it('should not display error for correct value', async () => {
        const input = screen.getByTestId('text-input-name');
        fireEvent.change(input, {
          target: { value: 'name-input-test-123a' },
        });
        fireEvent.blur(input);

        await waitFor(() =>
          Object.values(errors).forEach((val) => {
            expect(screen.queryByText(val)).not.toBeInTheDocument();
          })
        );
      });

      it('should display error for empty string', async () => {
        const input = screen.getByTestId('text-input-name') as HTMLInputElement;

        // Make field dirty first then clear
        fireEvent.change(input, {
          target: { value: 'a' },
        });
        fireEvent.blur(input);

        await waitFor(() =>
          expect(screen.queryByText(errors.MIN1_ERROR)).not.toBeInTheDocument()
        );

        fireEvent.change(input, {
          target: { value: '' },
        });
        fireEvent.blur(input);

        expect(input.value).toBe('');

        await waitFor(() =>
          expect(screen.getByText(errors.MIN1_ERROR)).toBeInTheDocument()
        );
      });

      it('should display error for a string too long', async () => {
        const nameInput = screen.getByTestId('text-input-name');

        // First enter a valid-length string (22 chars) — no error
        fireEvent.change(nameInput, {
          target: {
            value: 'abcdefghijklmnopqrstuv',
          },
        });
        fireEvent.blur(nameInput);

        await waitFor(() =>
          expect(screen.queryByText(errors.MAX22_ERROR)).not.toBeInTheDocument()
        );

        // Now enter a too-long string (24 chars) — error
        fireEvent.change(nameInput, {
          target: {
            value: 'abcdefghijklmnopqrstuvwx',
          },
        });
        fireEvent.blur(nameInput);

        await waitFor(() =>
          expect(screen.getByText(errors.MAX22_ERROR)).toBeInTheDocument()
        );
      });

      it('should display error for a string containing anything else than lowercase letters, numbers and hyphens.', async () => {
        const nameInput = screen.getByTestId('text-input-name');

        // Valid string — no error
        fireEvent.change(nameInput, {
          target: {
            value: 'test-123',
          },
        });
        fireEvent.blur(nameInput);

        await waitFor(() =>
          expect(
            screen.queryByText(errors.SPECIAL_CHAR_ERROR)
          ).not.toBeInTheDocument()
        );

        // Invalid string with special chars — error
        fireEvent.change(nameInput, {
          target: {
            value: 'test@123',
          },
        });
        fireEvent.blur(nameInput);

        await waitFor(() =>
          expect(
            screen.getByText(errors.SPECIAL_CHAR_ERROR)
          ).toBeInTheDocument()
        );
      });

      it('should display error for a string ending with a hyphen', async () => {
        const nameInput = screen.getByTestId('text-input-name');

        // Valid string — no error
        fireEvent.change(nameInput, {
          target: {
            value: 'test-123',
          },
        });
        fireEvent.blur(nameInput);

        await waitFor(() =>
          expect(
            screen.queryByText(errors.END_CHAR_ERROR)
          ).not.toBeInTheDocument()
        );

        // String ending with hyphen — error
        fireEvent.change(nameInput, {
          target: {
            value: 'test123-',
          },
        });
        fireEvent.blur(nameInput);

        await waitFor(() =>
          expect(screen.getByText(errors.END_CHAR_ERROR)).toBeInTheDocument()
        );
      });

      it('should display error for a string starting with a hyphen or number', async () => {
        const nameInput = screen.getByTestId('text-input-name');

        // Valid string — no error
        fireEvent.change(nameInput, {
          target: {
            value: 'test-123',
          },
        });
        fireEvent.blur(nameInput);

        await waitFor(() =>
          expect(
            screen.queryByText(errors.START_CHAR_ERROR)
          ).not.toBeInTheDocument()
        );

        // String starting with hyphen — error
        fireEvent.change(nameInput, {
          target: {
            value: '-test123',
          },
        });
        fireEvent.blur(nameInput);

        await waitFor(() =>
          expect(screen.getByText(errors.START_CHAR_ERROR)).toBeInTheDocument()
        );

        // String starting with number — error
        fireEvent.change(nameInput, {
          target: {
            value: '1test',
          },
        });
        fireEvent.blur(nameInput);

        await waitFor(() =>
          expect(screen.getByText(errors.START_CHAR_ERROR)).toBeInTheDocument()
        );
      });
    });
  });
};
