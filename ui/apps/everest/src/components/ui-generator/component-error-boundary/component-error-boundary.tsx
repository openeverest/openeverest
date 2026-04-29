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

import { Component, type ReactNode } from 'react';
import { Alert, AlertTitle } from '@mui/material';

interface ComponentErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
  componentName?: string;
}

interface ComponentErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

const initialState: ComponentErrorBoundaryState = {
  hasError: false,
  error: null,
};

export class ComponentErrorBoundary extends Component<
  ComponentErrorBoundaryProps,
  ComponentErrorBoundaryState
> {
  constructor(props: ComponentErrorBoundaryProps) {
    super(props);
    this.state = initialState;
  }

  static getDerivedStateFromError(error: Error): ComponentErrorBoundaryState {
    return { hasError: true, error };
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <Alert severity="error" sx={{ my: 1 }}>
          <AlertTitle>
            {this.props.componentName
              ? `Failed to render "${this.props.componentName}"`
              : 'Failed to render component'}
          </AlertTitle>
          {this.state.error?.message}
        </Alert>
      );
    }

    return this.props.children;
  }
}
