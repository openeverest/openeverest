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

import { PreviewContentText } from '../preview-section';
import { SectionProps } from './section.types';

export const PreviewSectionOne = ({
  dbName,
  topology,
  provider,
  k8sNamespace,
  instancePreset,
}: SectionProps) => {
  // Extract preset name from instancePreset (can be string or object)
  const presetName =
    typeof instancePreset === 'object' && instancePreset !== null
      ? instancePreset.label || instancePreset.value
      : instancePreset;

  return (
    <>
      {k8sNamespace && (
        <PreviewContentText text={`Namespace: ${k8sNamespace}`} />
      )}
      {provider && <PreviewContentText text={`Provider: ${provider}`} />}
      {dbName && <PreviewContentText text={`Name: ${dbName}`} />}
      {topology && topology?.type && (
        <PreviewContentText text={`Topology: ${topology?.type}`} />
      )}
      {presetName && <PreviewContentText text={`Template: ${presetName}`} />}
    </>
  );
};
