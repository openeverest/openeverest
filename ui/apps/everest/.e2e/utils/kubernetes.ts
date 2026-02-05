// everest
// Copyright (C) 2023 Percona LLC
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

import { execSync } from 'child_process';

export const getK8sUid = async () => {
  try {
    const command = `kubectl get namespace kube-system -o jsonpath='{.metadata.uid}'`;
    const output = execSync(command).toString();
    return output;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const getK8sObjectsNamespaceYaml = async (
  namespace: string,
  prefix: string,
  directory: string = 'upgrade-objects-yaml'
) => {
  const objects = ['pxc', 'psmdb', 'pg', 'sts', 'db', 'secret'];
  const command = `mkdir -p ${directory} || true`;
  execSync(command);

  for (const obj of objects) {
    const command = `kubectl get ${obj} -n ${namespace} -o jsonpath='{.items[*].metadata.name}' || true`;
    const output = execSync(command);
    const arr = output.toString().split(' ');
    for (const objName of arr) {
      const command = `kubectl get ${obj} ${objName} -n ${namespace} -oyaml > ${directory}/${prefix}-${obj}-${objName}.yaml || true`;
      execSync(command);
    }
  }
};

export const getK8sResource = async (
  resourceType: string,
  resourceName: string,
  namespace: string
) => {
  try {
    if (resourceType === 'postgresql') {
      resourceType = 'pg';
    }
    const command = `kubectl get --namespace ${namespace} ${resourceType} ${resourceName} -ojson`;
    const output = execSync(command);
    return JSON.parse(output.toString());
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const deleteAllK8sPVCs = async (namespace: string) => {
  try {
    const command = `kubectl delete --namespace ${namespace} pvc --all --timeout=30s || true`;
    execSync(command);
    console.log(`All PVCs deleted in namespace ${namespace}`);
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const getK8sNodes = async () => {
  try {
    const command = `kubectl get nodes --selector='!node-role.kubernetes.io/infra,!node-role.kubernetes.io/control-plane,!node-role.kubernetes.io/master' -o jsonpath='{.items[*].metadata.name}'`;
    const output = execSync(command).toString();
    return output.split(' ');
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const addLabelToK8sNode = async (
  nodeName: string,
  labelKey: string,
  labelValue: string
) => {
  try {
    const command = `kubectl label nodes ${nodeName} ${labelKey}=${labelValue}`;
    execSync(command);
    console.log(`Label ${labelKey}=${labelValue} added to node ${nodeName}`);
  } catch (error) {
    console.error(`Error adding label to node: ${error}`);
    throw error;
  }
};

export const removeLabelFromK8sNode = async (
  nodeName: string,
  labelKey: string
) => {
  try {
    const command = `kubectl label nodes ${nodeName} ${labelKey}-`;
    execSync(command);
    console.log(`Label ${labelKey} removed from node ${nodeName}`);
  } catch (error) {
    console.error(`Error removing label from node: ${error}`);
    throw error;
  }
};

export const removeFinalizersFromDB = async (
  resourceType: string,
  resourceName: string,
  namespace: string
) => {
  try {
    const command = `kubectl patch ${resourceType} ${resourceName} -n ${namespace} --type=json -p='[{"op": "remove", "path": "/metadata/finalizers"}]'`;
    execSync(command);
  } catch (error) {
    console.error(
      `Error removing finalizers from ${resourceType} ${resourceName}: ${error}`
    );
    throw error;
  }
};

export const getK8sProvider = async (): Promise<string> => {
  try {
    const versionCmd = `kubectl version -ojson`;
    const versionOutput = execSync(versionCmd).toString();
    const version = JSON.parse(versionOutput);

    const gitVersion: string = version?.serverVersion?.gitVersion ?? '';

    // Quick checks for GKE/EKS
    if (gitVersion.includes('gke')) {
      return 'GKE';
    }
    if (gitVersion.includes('eks')) {
      return 'EKS';
    }

    // Check for OpenShift APIs (ROSA is OpenShift on AWS)
    try {
      execSync(`kubectl api-resources --api-group=config.openshift.io`, {
        stdio: 'ignore',
      });
      return 'OpenShift';
    } catch {
      return 'unknown';
    }
  } catch (error) {
    console.error(`Error detecting Kubernetes provider: ${error}`);
    throw error;
  }
};
