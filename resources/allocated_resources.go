package resources

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

const instanceTypeLabel = "node.kubernetes.io/instance-type"

type Options struct {
	Labels                  string
	GetTotalsByInstanceType bool
	GetNodeDetails          bool
}

type ClusterMetrics struct {
	// Total contains the summed up metrics
	Totals NodeAllocatedResources `json:"totals,omitempty"`

	InstanceTypes []NodeAllocatedResources `json:"groupedby_instance_type,omitempty"`

	// Nodes contains allocated resources of each node
	Nodes []NodeAllocatedResources `json:"nodes,omitempty"`
}

// NodeAllocatedResources describes node allocated resources.
type NodeAllocatedResources struct {
	//Node name
	NodeCount int `json:"node_count,omitempty"`

	//Node name
	NodeName string `json:"node_name,omitempty"`

	// The instance Type like m5.2xlarge
	InstanceType string `json:"instance_type,omitempty"`

	// CPURequests is number of allocated milicores.
	CPURequests int64 `json:"cpu_requests"`

	// CPURequestsPercentage is the percentage of CPU, that is allocated.
	CPURequestsPercentage float64 `json:"cpu_requests_percentage"`

	// CPULimits is defined CPU limit.
	CPULimits int64 `json:"cpu_limits"`

	// CPULimitsPercentage is a percentage of defined CPU limit, can be over 100%, i.e.
	// overcommitted.
	CPULimitsPercentage float64 `json:"cpu_limits_percentage"`

	// CPUTotal is specified node CPU total in milicores.
	CPUTotal int64 `json:"cpu_total"`

	// MemoryRequests is a percentage of memory, that is allocated.
	MemoryRequests int64 `json:"memory_requests"`

	// MemoryRequestsPercentage is a percentage of memory, that is allocated.
	MemoryRequestsPercentage float64 `json:"memory_requests_percentage"`

	// MemoryLimits is defined memory limit.
	MemoryLimits int64 `json:"memory_limits"`

	// MemoryLimitsPercentage is a percentage of defined memory limit, can be over 100%, i.e.
	// overcommitted.
	MemoryLimitsPercentage float64 `json:"memory_limits_percentage"`

	// MemoryTotal is specified node memory total in bytes.
	MemoryTotal int64 `json:"memory_total"`

	// PodsAllocated in number of currently allocated pods on the node.
	PodsAllocated int `json:"pods_allocated"`

	// PodsTotal is maximum number of pods, that can be allocated on the node.
	PodsTotal int64 `json:"pods_total"`

	// PodsPercentage is a percentage of pods, that can be allocated on given node.
	PodsAllocatedPercentage float64 `json:"pods_allocated_percentage"`

	// Node Labels
	Labels map[string]string `json:"-"`
}

type AllocatedResourcesClient struct {
	kubeClient kubernetes.Interface
	options    *Options
}

func NewAllocatedResourcesClient(kubeClient kubernetes.Interface, options *Options) *AllocatedResourcesClient {
	app := new(AllocatedResourcesClient)
	app.kubeClient = kubeClient
	app.options = options
	return app
}

func (client *AllocatedResourcesClient) GetAllocatedResources() (ClusterMetrics, error) {

	var nodesAllocatedResources []NodeAllocatedResources
	var totalAllocatedResources NodeAllocatedResources
	var clusterMetrics ClusterMetrics

	instanceTypeMap := make(map[string]NodeAllocatedResources)

	nodeFieldSelector, err := fields.ParseSelector(client.options.Labels)
	if err != nil {
		return clusterMetrics, err
	}

	nodes, err := client.kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{LabelSelector: nodeFieldSelector.String()})
	if err != nil {
		return clusterMetrics, err
	}

	for _, node := range nodes.Items {

		instanceTypeMap[node.Labels[instanceTypeLabel]] = NodeAllocatedResources{}

		podFieldSelector, err := fields.ParseSelector("spec.nodeName=" + node.Name + ",status.phase!=" + string(v1.PodSucceeded) + ",status.phase!=" + string(v1.PodFailed))
		if err != nil {
			return clusterMetrics, err
		}
		nodeNonTerminatedPodsList, err := client.kubeClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{FieldSelector: podFieldSelector.String()})
		if err != nil {
			return clusterMetrics, err
		}

		nodeAllocatedResources, err := getNodeAllocatedResources(node, nodeNonTerminatedPodsList)
		if err != nil {
			return clusterMetrics, err
		}

		nodesAllocatedResources = append(nodesAllocatedResources, nodeAllocatedResources)
	}

	// Calculate totals
	for _, node := range nodesAllocatedResources {
		totalAllocatedResources.CPUTotal += node.CPUTotal
		totalAllocatedResources.CPURequests += node.CPURequests
		totalAllocatedResources.CPULimits += node.CPULimits
		totalAllocatedResources.MemoryTotal += node.MemoryTotal
		totalAllocatedResources.MemoryRequests += node.MemoryRequests
		totalAllocatedResources.MemoryLimits += node.MemoryLimits
		totalAllocatedResources.PodsAllocated += node.PodsAllocated
		totalAllocatedResources.PodsTotal += node.PodsTotal
	}

	if len(nodesAllocatedResources) > 0 {
		totalAllocatedResources.NodeCount = len(nodesAllocatedResources)
		totalAllocatedResources.CPURequestsPercentage = float64(totalAllocatedResources.CPURequests * 100 / totalAllocatedResources.CPUTotal)
		totalAllocatedResources.CPULimitsPercentage = float64(totalAllocatedResources.CPULimits * 100 / totalAllocatedResources.CPUTotal)
		totalAllocatedResources.MemoryRequestsPercentage = float64(totalAllocatedResources.MemoryRequests * 100 / totalAllocatedResources.MemoryTotal)
		totalAllocatedResources.MemoryLimitsPercentage = float64(totalAllocatedResources.MemoryLimits * 100 / totalAllocatedResources.MemoryTotal)
		totalAllocatedResources.PodsAllocatedPercentage = float64(int64(totalAllocatedResources.PodsAllocated) * 100 / totalAllocatedResources.PodsTotal)
	}
	clusterMetrics.Totals = totalAllocatedResources

	// Calculate totals grouped by the instance type
	if client.options.GetTotalsByInstanceType {
		instanceTypeAllocatedResourcesSlice := []NodeAllocatedResources{}
		for instanceType, instanceTypeAllocatedResources := range instanceTypeMap {
			for _, node := range nodesAllocatedResources {
				if node.Labels[instanceTypeLabel] == instanceType {
					instanceTypeAllocatedResources.NodeCount += 1
					instanceTypeAllocatedResources.CPUTotal += node.CPUTotal
					instanceTypeAllocatedResources.CPURequests += node.CPURequests
					instanceTypeAllocatedResources.CPULimits += node.CPULimits
					instanceTypeAllocatedResources.MemoryTotal += node.MemoryTotal
					instanceTypeAllocatedResources.MemoryRequests += node.MemoryRequests
					instanceTypeAllocatedResources.MemoryLimits += node.MemoryLimits
					instanceTypeAllocatedResources.PodsAllocated += node.PodsAllocated
					instanceTypeAllocatedResources.PodsTotal += node.PodsTotal
				}
			}

			if len(nodesAllocatedResources) > 0 {
				instanceTypeAllocatedResources.InstanceType = instanceType
				instanceTypeAllocatedResources.CPURequestsPercentage = float64(instanceTypeAllocatedResources.CPURequests * 100 / instanceTypeAllocatedResources.CPUTotal)
				instanceTypeAllocatedResources.CPULimitsPercentage = float64(instanceTypeAllocatedResources.CPULimits * 100 / instanceTypeAllocatedResources.CPUTotal)
				instanceTypeAllocatedResources.MemoryRequestsPercentage = float64(instanceTypeAllocatedResources.MemoryRequests * 100 / instanceTypeAllocatedResources.MemoryTotal)
				instanceTypeAllocatedResources.MemoryLimitsPercentage = float64(instanceTypeAllocatedResources.MemoryLimits * 100 / instanceTypeAllocatedResources.MemoryTotal)
				instanceTypeAllocatedResources.PodsAllocatedPercentage = float64(int64(instanceTypeAllocatedResources.PodsAllocated) * 100 / instanceTypeAllocatedResources.PodsTotal)
			}
			instanceTypeAllocatedResourcesSlice = append(instanceTypeAllocatedResourcesSlice, instanceTypeAllocatedResources)
		}
		clusterMetrics.InstanceTypes = instanceTypeAllocatedResourcesSlice
	}

	// Return node details
	if client.options.GetNodeDetails {
		clusterMetrics.Nodes = nodesAllocatedResources
	}

	return clusterMetrics, nil
}

func podRequestsAndLimits(pod *v1.Pod) (reqs map[v1.ResourceName]resource.Quantity, limits map[v1.ResourceName]resource.Quantity, err error) {
	reqs, limits = map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}
	for _, container := range pod.Spec.Containers {
		for name, quantity := range container.Resources.Requests {
			if value, ok := reqs[name]; !ok {
				reqs[name] = *quantity.ToDec()
			} else {
				value.Add(quantity)
				reqs[name] = value
			}
		}
		for name, quantity := range container.Resources.Limits {
			if value, ok := limits[name]; !ok {
				limits[name] = *quantity.ToDec()
			} else {
				value.Add(quantity)
				limits[name] = value
			}
		}
	}
	return
}

func getNodeAllocatedResources(node v1.Node, podList *v1.PodList) (NodeAllocatedResources, error) {
	reqs, limits := map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}

	for _, pod := range podList.Items {

		podReqs, podLimits, err := podRequestsAndLimits(&pod)
		if err != nil {
			return NodeAllocatedResources{}, err
		}
		for podReqName, podReqValue := range podReqs {
			if value, ok := reqs[podReqName]; !ok {
				reqs[podReqName] = *podReqValue.ToDec()
			} else {
				value.Add(podReqValue)
				reqs[podReqName] = value
			}
		}
		for podLimitName, podLimitValue := range podLimits {
			if value, ok := limits[podLimitName]; !ok {
				limits[podLimitName] = *podLimitValue.ToDec()
			} else {
				value.Add(podLimitValue)
				limits[podLimitName] = value
			}
		}
	}

	cpuRequests, cpuLimits, memoryRequests, memoryLimits := reqs[v1.ResourceCPU],
		limits[v1.ResourceCPU], reqs[v1.ResourceMemory], limits[v1.ResourceMemory]

	var cpuRequestsPercentage, cpuLimitsPercentage float64 = 0, 0
	if total := float64(node.Status.Capacity.Cpu().MilliValue()); total > 0 {
		cpuRequestsPercentage = float64(cpuRequests.MilliValue()) / total * 100
		cpuLimitsPercentage = float64(cpuLimits.MilliValue()) / total * 100
	}

	var memoryRequestsPercentage, memoryLimitsPercentage float64 = 0, 0
	if total := float64(node.Status.Capacity.Memory().MilliValue()); total > 0 {
		memoryRequestsPercentage = float64(memoryRequests.MilliValue()) / total * 100
		memoryLimitsPercentage = float64(memoryLimits.MilliValue()) / total * 100
	}

	var podsAllocatedPercentage float64 = 0
	var podsTotal int64 = node.Status.Capacity.Pods().Value()
	if podsTotal > 0 {
		podsAllocatedPercentage = float64(len(podList.Items)) / float64(podsTotal) * 100
	}

	return NodeAllocatedResources{
		Labels:                node.Labels,
		NodeName:              node.Name,
		InstanceType:          node.Labels[instanceTypeLabel],
		CPUTotal:              node.Status.Capacity.Cpu().MilliValue(),
		CPURequests:           cpuRequests.MilliValue(),
		CPURequestsPercentage: cpuRequestsPercentage,
		CPULimits:             cpuLimits.MilliValue(),
		CPULimitsPercentage:   cpuLimitsPercentage,

		MemoryTotal:              node.Status.Capacity.Memory().Value(),
		MemoryRequests:           memoryRequests.Value(),
		MemoryRequestsPercentage: memoryRequestsPercentage,
		MemoryLimits:             memoryLimits.Value(),
		MemoryLimitsPercentage:   memoryLimitsPercentage,

		PodsAllocated:           len(podList.Items),
		PodsTotal:               podsTotal,
		PodsAllocatedPercentage: podsAllocatedPercentage,
	}, nil
}
