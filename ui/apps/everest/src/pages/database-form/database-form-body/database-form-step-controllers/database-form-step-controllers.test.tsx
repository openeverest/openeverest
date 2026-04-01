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

import { fireEvent, render, screen } from '@testing-library/react';
import DatabaseFormStepControllers from './database-form-step-controllers';

describe('DatabaseFormStepControllers', () => {
  it('does not trigger native form submission when cancel is clicked', () => {
    const handleFormSubmit = vi.fn(
      (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
      }
    );
    const handleCancel = vi.fn();

    render(
      <form onSubmit={handleFormSubmit}>
        <DatabaseFormStepControllers
          disableBack
          disableSubmit={false}
          disableCancel={false}
          disableNext={false}
          showSubmit
          onPreviousClick={vi.fn()}
          onNextClick={vi.fn()}
          onCancel={handleCancel}
          onSubmit={vi.fn()}
        />
      </form>
    );

    fireEvent.click(screen.getByTestId('db-wizard-cancel-button'));

    expect(handleCancel).toHaveBeenCalledTimes(1);
    expect(handleFormSubmit).not.toHaveBeenCalled();
  });

  it('does not trigger native form submission when submit is clicked', () => {
    const handleNativeFormSubmit = vi.fn(
      (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
      }
    );
    const handleSubmit = vi.fn();

    render(
      <form onSubmit={handleNativeFormSubmit}>
        <DatabaseFormStepControllers
          disableBack
          disableSubmit={false}
          disableCancel={false}
          disableNext={false}
          showSubmit
          onPreviousClick={vi.fn()}
          onNextClick={vi.fn()}
          onCancel={vi.fn()}
          onSubmit={handleSubmit}
        />
      </form>
    );

    fireEvent.click(screen.getByTestId('db-wizard-submit-button'));

    // Submit must NOT trigger a native form submission event.
    // If it does, React Router's data router can intercept it as a navigation,
    // which fires the useBlocker and shows the cancel dialog.
    expect(handleNativeFormSubmit).not.toHaveBeenCalled();
    expect(handleSubmit).toHaveBeenCalledTimes(1);
  });

  it('calls onSubmit exactly once when submit is clicked', () => {
    const handleSubmit = vi.fn();

    render(
      <form>
        <DatabaseFormStepControllers
          disableBack
          disableSubmit={false}
          disableCancel={false}
          disableNext={false}
          showSubmit
          onPreviousClick={vi.fn()}
          onNextClick={vi.fn()}
          onCancel={vi.fn()}
          onSubmit={handleSubmit}
        />
      </form>
    );

    fireEvent.click(screen.getByTestId('db-wizard-submit-button'));

    expect(handleSubmit).toHaveBeenCalledTimes(1);
  });
});
