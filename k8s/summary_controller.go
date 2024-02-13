package k8s

import (
	"sync"
	"strings"
	"regexp"
	"context"
	"time"
	"os/exec"

	"github.com/pjy0381/ktop/views/model"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsV1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

var (
    clientset *kubernetes.Clientset
    lastUpdateTime time.Time
    updateInterval = 5 * time.Second
)

func (c *Controller) setupSummaryHandler(ctx context.Context, handlerFunc RefreshSummaryFunc) {
	go func() {
		c.refreshSummary(ctx, handlerFunc)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.refreshSummary(ctx, handlerFunc); err != nil {
					continue
				}
			}
		}
	}()
}

func (c *Controller) refreshSummary(ctx context.Context, handlerFunc RefreshSummaryFunc) error {
	var summary model.ClusterSummary

	// extract namespace summary
	namespaces, err := c.GetNamespaceList(ctx)
	if err != nil {
		return err
	}
	summary.Namespaces = len(namespaces)

	nodes, err := c.GetNodeList(ctx)
	if err != nil {
		return err
	}
	summary.Uptime = metav1.NewTime(time.Now())
	summary.NodesCount = len(nodes)
	summary.AllocatableNodeMemTotal = resource.NewQuantity(0, resource.DecimalSI)
	summary.AllocatableNodeCpuTotal = resource.NewQuantity(0, resource.DecimalSI)
	summary.UsageNodeMemTotal = resource.NewQuantity(0, resource.DecimalSI)
	summary.UsageNodeCpuTotal = resource.NewQuantity(0, resource.DecimalSI)
	for _, node := range nodes {
		if model.GetNodeReadyStatus(node) == string(coreV1.NodeReady) {
			summary.NodesReady++
		}
		if node.CreationTimestamp.Before(&summary.Uptime) {
			summary.Uptime = node.CreationTimestamp
		}

		summary.Pressures += len(model.GetNodePressures(node))
		summary.ImagesCount += len(node.Status.Images)
		summary.VolumesInUse += len(node.Status.VolumesInUse)

		summary.AllocatableNodeMemTotal.Add(*node.Status.Allocatable.Memory())
		summary.AllocatableNodeCpuTotal.Add(*node.Status.Allocatable.Cpu())

		metrics, err := c.GetNodeMetrics(ctx, node.Name)
		if err != nil {
			metrics = new(metricsV1beta1.NodeMetrics)
		}
		summary.UsageNodeMemTotal.Add(*metrics.Usage.Memory())
		summary.UsageNodeCpuTotal.Add(*metrics.Usage.Cpu())

	}

	// extract pods summary
	pods, err := c.GetPodList(ctx)
	if err != nil {
		return err
	}
	summary.PodsAvailable = 0
	summary.RequestedPodMemTotal = resource.NewQuantity(0, resource.DecimalSI)
	summary.RequestedPodCpuTotal = resource.NewQuantity(0, resource.DecimalSI)

	nodeMetricsCache := make(map[string]*metricsV1beta1.NodeMetrics)
	for _, pod := range pods {
		// retrieve metrics per pod
                podMetrics, err := c.GetPodMetricsByName(ctx, pod)
                if err != nil {
                        podMetrics = new(metricsV1beta1.PodMetrics)
                }
                // retrieve and cache node metrics for related pod-node
                if metrics, ok := nodeMetricsCache[pod.Spec.NodeName]; !ok {
                        metrics, err = c.GetNodeMetrics(ctx, pod.Spec.NodeName)
                        if err != nil {
                                metrics = new(metricsV1beta1.NodeMetrics)
                        }
                        nodeMetricsCache[pod.Spec.NodeName] = metrics
                }
                nodeMetrics := nodeMetricsCache[pod.Spec.NodeName]
                podModel := model.NewPodModel(pod, podMetrics, nodeMetrics)

		if podModel.Status == "Completed" {
			continue
		}

		summary.PodsAvailable++
		if pod.Status.Phase == coreV1.PodRunning && podModel.ReadyContainers == podModel.TotalContainers {
			summary.PodsRunning++
		}
		containerSummary := model.GetPodContainerSummary(pod)
		summary.RequestedPodMemTotal.Add(*containerSummary.RequestedMemQty)
		summary.RequestedPodCpuTotal.Add(*containerSummary.RequestedCpuQty)

		// etcd count
		if pod.Labels["component"] == "etcd" {
			summary.EtcdCount++
			if pod.Status.Phase == coreV1.PodRunning {
				summary.EtcdReady++
			}
		}
	}

	// deployments count
	deps, err := c.GetDeploymentList(ctx)
	if err != nil {
		return err
	}
	for _, dep := range deps {
		summary.DeploymentsTotal += int(dep.Status.Replicas)
		summary.DeploymentsReady += int(dep.Status.ReadyReplicas)
	}

	// deamonset count
	daemonsets, err := c.GetDaemonSetList(ctx)
	if err != nil {
		return err
	}
	for _, set := range daemonsets {
		summary.DaemonSetsDesired += int(set.Status.DesiredNumberScheduled)
		summary.DaemonSetsReady += int(set.Status.NumberReady)
	}

	// replicasets count
	replicasets, err := c.GetReplicaSetList(ctx)
	if err != nil {
		return err
	}
	for _, replica := range replicasets {
		summary.ReplicaSetsDesired += int(replica.Status.Replicas)
		summary.ReplicaSetsReady += int(replica.Status.ReadyReplicas)
	}

	// statefulsets count
	statefulsets, err := c.GetStatefulSetList(ctx)
	if err != nil {
		return err
	}
	for _, stateful := range statefulsets {
		summary.StatefulSetsReady += int(stateful.Status.ReadyReplicas)
	}

	// extract jobs summary
	jobs, err := c.GetJobList(ctx)
	if err != nil {
		return err
	}
	summary.JobsCount = len(jobs)
	cronjobs, err := c.GetCronJobList(ctx)
	if err != nil {
		return err
	}
	summary.CronJobsCount = len(cronjobs)

	pvs, err := c.GetPVList(ctx)
	if err != nil {
		return err
	}
	summary.PVCount = len(pvs)
	summary.PVsTotal = resource.NewQuantity(0, resource.DecimalSI)
	for _, pv := range pvs {
		if pv.Status.Phase == coreV1.VolumeBound {
			summary.PVsTotal.Add(*pv.Spec.Capacity.Storage())
		}
	}

	pvcs, err := c.GetPVCList(ctx)
	if err != nil {
		return err
	}
	summary.PVCCount = len(pvcs)
	summary.PVCsTotal = resource.NewQuantity(0, resource.DecimalSI)
	for _, pvc := range pvcs {
		if pvc.Status.Phase == coreV1.ClaimBound {
			summary.PVCsTotal.Add(*pvc.Spec.Resources.Requests.Storage())
		}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	// count service
	for _, node := range nodes {
		summary.KubeletCount++
		summary.ContainerdCount++
		summary.SciniCount++
		nodeInfo := node.Status.NodeInfo

		// kubelet
                if isKubeletHealthy(node) {
                        summary.KubeletReady++
                }

		// containerd
		if len(removeNumbersAndDotRegex(nodeInfo.ContainerRuntimeVersion)) != 0 {
			summary.ContainerdReady++
                }

		// scini
		wg.Add(1)
		go func(node *coreV1.Node) {
			defer wg.Done()
			status := getKubeletStatus(node.Status.Addresses[0].Address)
			mu.Lock()
			defer mu.Unlock()
			if status == "active" {
				summary.SciniReady++
			}
		}(node)

        }

	wg.Wait()
	handlerFunc(ctx, summary)
	return nil
}

func isKubeletHealthy(node *coreV1.Node) bool {
        for _, condition := range node.Status.Conditions {
		if condition.Status == coreV1.ConditionTrue {
			return true
		}
        }
        return false
}

func removeNumbersAndDotRegex(input string) string {
	re := regexp.MustCompile(`[^0-9]`)
	return re.ReplaceAllString(input, "")
}

func getKubeletStatus(node string) string {
    cmd := exec.Command("ssh", "-o StrictHostKeyChecking=no",  node, "sudo", "systemctl", "status", "scini")
    // 결과에서 상태 부분 추출
    output, err := cmd.Output()
    if err != nil {
        return ""
    }
    status := extractStatus(string(output))
    if status == "" {
        return ""
    }

    defer cmd.Process.Kill()
    return status
}

func extractStatus(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Active:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// "Active:" 다음에 상태가 오므로, 그 다음에 있는 단어가 상태
				return fields[1]
			}
		}
	}
	return ""
}

