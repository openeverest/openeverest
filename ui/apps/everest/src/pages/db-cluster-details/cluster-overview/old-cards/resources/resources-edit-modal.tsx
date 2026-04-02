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

// @ts-nocheck
// TODO remove this file after release of v2
import { DbType } from '@percona/types';
import { SubmitHandler } from 'react-hook-form';
import { z } from 'zod';
import { ResourcesForm } from 'components/cluster-form';
import { FormDialog } from 'components/form-dialog';
import { useKubernetesClusterInfo } from 'hooks';

type Props = {
  handleCloseModal: () => void;
  dbType: DbType;
  shardingEnabled: boolean;
  onSubmit: SubmitHandler<z.infer<ReturnType<typeof resourcesFormSchema>>>;
  defaultValues: z.infer<ReturnType<typeof resourcesFormSchema>>;
  storageClass: string;
  allowDiskDescaling: boolean;
};

const ResourcesEditModal = ({
  handleCloseModal,
  dbType,
  shardingEnabled,
  onSubmit,
  defaultValues,
  storageClass,
  allowDiskDescaling,
}: Props) => {
  const { data: clusterInfo } = useKubernetesClusterInfo([
    'resources-cluster-info',
  ]);
  const allowVolumeExpansion = (clusterInfo?.storageClasses || []).find(
    (item) => item.metadata.name === storageClass
  )?.allowVolumeExpansion;

  return (
    <FormDialog
      dataTestId="edit-resources"
      size="XXXL"
      isOpen
      closeModal={handleCloseModal}
      headerMessage="Edit Topology"
      submitMessage="Save"
      schema={resourcesFormSchema(
        defaultValues,
        false,
        false,
        allowDiskDescaling
      )}
      onSubmit={onSubmit}
      defaultValues={defaultValues}
    >
      <ResourcesForm
        dbType={dbType}
        pairProxiesWithNodes={false}
        showSharding={dbType === DbType.Mongo}
        disableDiskInput={!allowVolumeExpansion}
        defaultValues={defaultValues}
        allowDiskInputUpdate={false}
        hideProxies={dbType === DbType.Mongo && !shardingEnabled}
      />
    </FormDialog>
  );
};

export default ResourcesEditModal;
