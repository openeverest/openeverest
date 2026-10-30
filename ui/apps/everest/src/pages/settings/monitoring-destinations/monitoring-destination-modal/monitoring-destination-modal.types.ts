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

import { OpenApiParameterFields } from 'components/openapi-parameters-fields';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { MonitoringSectionForm } from 'utils/monitoring-classes';

export interface MonitoringDestinationFormValues {
  name: string;
  namespace: string;
  className: string;
  authMode: string;
  replaceCredentials: boolean;
  credentialsSecretName: string;
  credentials: Record<string, unknown>;
  spec: Record<string, unknown>;
  fallbackParameters: Record<string, unknown>;
  insecureSkipVerify: boolean;
  caSecretName: string;
  caSecretKey: string;
}

// The forms a class renders for a destination, for the chosen auth mode.
export interface DestinationClassForms {
  destination?: MonitoringSectionForm;
  // Used when the class ships no destination form.
  parameterFields: OpenApiParameterFields;
  credentials?: MonitoringSectionForm;
}

export interface MonitoringDestinationTarget {
  namespace: string;
  name: string;
}

export interface MonitoringDestinationModalProps {
  // The destination to edit; a new one is created without it.
  target?: MonitoringDestinationTarget;
  classes: MonitoringClass[];
  // Namespaces the user may create destinations in.
  namespaces: string[];
  onClose: () => void;
}

export interface MonitoringDestinationFormProps {
  original?: MonitoringDestination;
  classes: MonitoringClass[];
  namespaces: string[];
  // Set after a save was refused because the destination changed meanwhile.
  changedMeanwhile: boolean;
  onChangedMeanwhile: () => void;
  onClose: () => void;
}
