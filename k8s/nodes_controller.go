package k8s

import (
		"sync"
		"context"
		"fmt"
		"time"

		"github.com/pjy0381/ktop/views/model"
		coreV1 "k8s.io/api/core/v1"
		"k8s.io/apimachinery/pkg/api/resource"
		"k8s.io/apimachinery/pkg/labels"
		metricsV1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
       )

func (c *Controller) GetNode(ctx context.Context, nodeName string) (*coreV1.Node, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	node, err := c.nodeInformer.Lister().Get(nodeName)
		if err != nil {
			return nil, err
		}
	return node, nil
}

func (c *Controller) GetNodeList(ctx context.Context) ([]*coreV1.Node, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if err := c.assertNodeAuthz(ctx); err != nil {
		return nil, err
	}

	items, err := c.nodeInformer.Lister().List(labels.Everything())
		if err != nil {
			return nil, err
		}
	return items, nil
}

func (c *Controller) GetNodeModels(ctx context.Context) (models []model.NodeModel, err error) {
	nodes, err := c.GetNodeList(ctx)
		if err != nil {
			return
		}
	// map each node to their pods
	pods, err := c.GetPodList(ctx)
		if err != nil {
			return nil, err
		}

	var mu sync.Mutex
	var wg sync.WaitGroup

	nodeStatusMap := make(map[string]string)
        for _, node := range nodes {
		wg.Add(1)
		go func(node *coreV1.Node) {
			defer wg.Done()
			status := getKubeletStatus(GetNodeIp(node, coreV1.NodeInternalIP), "scini")
			mu.Lock()
			defer mu.Unlock()
			nodeStatusMap[node.Name] = status
		}(node)
	}
	wg.Wait()

	for _, node := range nodes {
		metrics, err := c.GetNodeMetrics(ctx, node.Name)
		if err != nil {
			metrics = new(metricsV1beta1.NodeMetrics)
		}
		nodePods := getPodNodes(node.Name, pods)
		podsCount := len(nodePods)
		nodeModel := model.NewNodeModel(node, metrics)
		nodeModel.PodsCount = podsCount
		nodeModel.RequestedPodMemQty = resource.NewQuantity(0, resource.DecimalSI)
		nodeModel.RequestedPodCpuQty = resource.NewQuantity(0, resource.DecimalSI)
		for _, pod := range nodePods {
			summary := model.GetPodContainerSummary(pod)
			nodeModel.RequestedPodMemQty.Add(*summary.RequestedMemQty)
			nodeModel.RequestedPodCpuQty.Add(*summary.RequestedCpuQty)
		}

		nodeModel.Kubelet = isKubeletHealthy(node)
		nodeModel.Containerd = len(removeNumbersAndDotRegex(node.Status.NodeInfo.ContainerRuntimeVersion)) != 0

		status := nodeStatusMap[node.Name]
		nodeModel.Scini = (status == "active")

		models = append(models, *nodeModel)
	}

	return
}

func (c *Controller) assertNodeAuthz(ctx context.Context) error {
	authzd, err := c.client.IsAuthz(ctx, "nodes", []string{"get", "list"})
	if err != nil {
		return fmt.Errorf("failed to check node authorization: %w", err)
	}
	if !authzd {
		return fmt.Errorf("node get, list not authorized")
	}
	return nil
}

func (c *Controller) setupNodeHandler(ctx context.Context, handlerFunc RefreshNodesFunc) {
	go func() {
		c.refreshNodes(ctx, handlerFunc) // initial refresh
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.refreshNodes(ctx, handlerFunc); err != nil {
					continue
				}
			}
		}
	}()
}

func (c *Controller) refreshNodes(ctx context.Context, handlerFunc RefreshNodesFunc) error {
	models, err := c.GetNodeModels(ctx)
	if err != nil {
		return err
	}
	handlerFunc(ctx, models)
	return nil
}

func getPodNodes(nodeName string, pods []*coreV1.Pod) []*coreV1.Pod {
	var result []*coreV1.Pod

	for _, pod := range pods {
		if pod.Spec.NodeName == nodeName {
			result = append(result, pod)
		}
	}
	return result
}

func GetNodeIp(node *coreV1.Node, addrType coreV1.NodeAddressType) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == addrType {
			return addr.Address
		}
	}
	return "<none>"
}
