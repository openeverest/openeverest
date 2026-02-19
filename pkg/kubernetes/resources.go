package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/convertors"
)

const (
	// Max size of volume for AWS Elastic Block Storage service is 16TiB.
	maxVolumeSizeEBS = 16 * 1024 * 1024 * 1024 * 1024

	// 39 is a default limit for EKS cluster nodes ...
	eksVolumeLimitPerNode = 39
)

// GetAllClusterResources goes through all cluster nodes and sums their allocatable resources.
func (k *Kubernetes) GetAllClusterResources(
	ctx context.Context, clusterType ClusterType, volumes *corev1.PersistentVolumeList,
) (uint64, uint64, uint64, error) {
	cpuMillis, memoryBytes, diskSizeBytes, volumeCountEKS, err := k.getResourcesFromNodes(ctx, clusterType)
	if err != nil {
		return 0, 0, 0, err
	}

	if clusterType == ClusterTypeEKS {
		volumeCountEKSBackup := volumeCountEKS
		volumeCountEKS -= uint64(len(volumes.Items))
		if volumeCountEKS > volumeCountEKSBackup {
			// handle uint underflow
			volumeCountEKS = 0
		}

		consumedBytes, err := sumVolumesSize(volumes)
		if err != nil {
			return 0, 0, 0, errors.Join(err, errors.New("failed to sum persistent volumes storage sizes"))
		}
		diskSizeBytes = (volumeCountEKS * maxVolumeSizeEBS) + consumedBytes
	}
	return cpuMillis, memoryBytes, diskSizeBytes, nil
}

func (k *Kubernetes) getResourcesFromNodes(ctx context.Context, clusterType ClusterType) (uint64, uint64, uint64, uint64, error) {
	var cpuMillis, memoryBytes, diskSizeBytes uint64

	nodes, err := k.ListWorkerNodes(ctx)
	if err != nil || nodes == nil {
		return 0, 0, 0, 0, errors.Join(err, errors.New("could not get a list of nodes"))
	}
	var volumeCountEKS uint64
	for _, node := range nodes.Items {
		cpu, memory, err := getResources(node.Status.Allocatable)
		if err != nil {
			return 0, 0, 0, 0, errors.Join(err, errors.New("could not get allocatable resources of the node"))
		}
		cpuMillis += cpu
		memoryBytes += memory

		switch clusterType {
		case ClusterTypeUnknown:
			return 0, 0, 0, 0, errors.New("unknown cluster type")
		case ClusterTypeGeneric, ClusterTypeOpenShift, ClusterTypeGKE:
			// TODO support other cluster types
			continue
		case ClusterTypeMinikube:
			bytes, err := k.getMinikubeDiskSizeBytes(node)
			if err != nil {
				return 0, 0, 0, 0, err
			}
			diskSizeBytes += bytes
		case ClusterTypeEKS:
			v, err := k.getEKSVolumeCount(node)
			if err != nil {
				return 0, 0, 0, 0, err
			}
			volumeCountEKS += v
		}
	}

	return cpuMillis, memoryBytes, diskSizeBytes, volumeCountEKS, nil
}

func (k *Kubernetes) getEKSVolumeCount(node corev1.Node) (uint64, error) {
	var volumeCountEKS uint64

	// See https://kubernetes.io/docs/tasks/administer-cluster/out-of-resource/#scheduler.
	if IsNodeInCondition(node, corev1.NodeDiskPressure) {
		return 0, nil
	}

	// Get node's type.
	nodeType, ok := node.Labels["beta.kubernetes.io/instance-type"]
	if !ok {
		return 0, errors.New("dealing with AWS EKS cluster but the node does not have label 'beta.kubernetes.io/instance-type'")
	}
	volumeLimitPerNode := uint64(eksVolumeLimitPerNode)
	typeAndSize := strings.Split(strings.ToLower(nodeType), ".")
	if len(typeAndSize) < 2 { //nolint:mnd
		return 0, fmt.Errorf("failed to parse EKS node type '%s', it's not in expected format 'type.size'", nodeType)
	}
	// ... however, if the node type is one of M5, C5, R5, T3, Z1D it's 25.
	limitedVolumesSet := map[string]struct{}{
		"m5": {}, "c5": {}, "r5": {}, "t3": {}, "t1d": {},
	}
	if _, ok := limitedVolumesSet[typeAndSize[0]]; ok {
		volumeLimitPerNode = 25
	}
	volumeCountEKS += volumeLimitPerNode

	return volumeCountEKS, nil
}

func (k *Kubernetes) getMinikubeDiskSizeBytes(node corev1.Node) (uint64, error) {
	storage, ok := node.Status.Allocatable[corev1.ResourceEphemeralStorage]
	if !ok {
		return 0, errors.New("could not get storage size of the node")
	}
	bytes, err := convertors.StrToBytes(storage.String())
	if err != nil {
		return 0, errors.Join(err, fmt.Errorf("could not convert storage size '%s' to bytes", storage.String()))
	}
	return bytes, err
}

// getResources extracts resources out of corev1.ResourceList and converts them to int64 values.
// Millicpus are used for CPU values and bytes for memory.
func getResources(resources corev1.ResourceList) (cpuMillis uint64, memoryBytes uint64, err error) { //nolint:nonamedreturns
	cpu, ok := resources[corev1.ResourceCPU]
	if ok {
		cpuMillis, err = convertors.StrToMilliCPU(cpu.String())
		if err != nil {
			return 0, 0, errors.Join(err, fmt.Errorf("failed to convert '%s' to millicpus", cpu.String()))
		}
	}
	memory, ok := resources[corev1.ResourceMemory]
	if ok {
		memoryBytes, err = convertors.StrToBytes(memory.String())
		if err != nil {
			return 0, 0, errors.Join(err, fmt.Errorf("failed to convert '%s' to bytes", memory.String()))
		}
	}
	return cpuMillis, memoryBytes, nil
}

// sumVolumesSize returns sum of persistent volumes storage size in bytes.
func sumVolumesSize(pvs *corev1.PersistentVolumeList) (sum uint64, err error) { //nolint:nonamedreturns
	for _, pv := range pvs.Items {
		bytes, err := convertors.StrToBytes(pv.Spec.Capacity.Storage().String())
		if err != nil {
			return 0, err
		}
		sum += bytes
	}
	return
}

// GetConsumedCPUAndMemory returns consumed CPU and Memory in given namespace. If namespace
// is empty, it tries to get them from all namespaces.
func (k *Kubernetes) GetConsumedCPUAndMemory(ctx context.Context, namespace string) ( //nolint:nonamedreturns
	cpuMillis uint64, memoryBytes uint64, err error,
) {
	// Get CPU and Memory Requests of Pods' containers.
	pods, err := k.ListPods(ctx, ctrlclient.InNamespace(namespace))
	if err != nil {
		return 0, 0, errors.Join(err, errors.New("failed to get consumed resources"))
	}
	for _, ppod := range pods.Items {
		if ppod.Status.Phase != corev1.PodRunning {
			continue
		}
		nonTerminatedInitContainers := make([]corev1.Container, 0, len(ppod.Spec.InitContainers))
		for _, container := range ppod.Spec.InitContainers {
			if !IsContainerInState(
				ppod.Status.InitContainerStatuses, ContainerStateTerminated,
			) {
				nonTerminatedInitContainers = append(nonTerminatedInitContainers, container)
			}
		}
		for _, container := range append(ppod.Spec.Containers, nonTerminatedInitContainers...) {
			cpu, memory, err := getResources(container.Resources.Requests)
			if err != nil {
				return 0, 0, errors.Join(err, errors.New("failed to sum all consumed resources"))
			}
			cpuMillis += cpu
			memoryBytes += memory
		}
	}

	return cpuMillis, memoryBytes, nil
}

// GetConsumedDiskBytes returns consumed bytes. The strategy differs based on k8s cluster type.
func (k *Kubernetes) GetConsumedDiskBytes(
	_ context.Context, clusterType ClusterType, volumes *corev1.PersistentVolumeList,
) (uint64, error) {
	switch clusterType {
	case ClusterTypeUnknown:
		return 0, errors.New("unknown cluster type")
	case ClusterTypeGeneric, ClusterTypeOpenShift, ClusterTypeGKE:
		// TODO support other cluster types.
		return 0, nil
	case ClusterTypeMinikube:
		// Minikube does not seem to have support for this.
		return 0, nil
	case ClusterTypeEKS:
		consumedBytes, err := sumVolumesSize(volumes)
		if err != nil {
			return 0, errors.Join(err, errors.New("failed to sum persistent volumes storage sizes"))
		}
		return consumedBytes, nil
	}

	return 0, nil
}
