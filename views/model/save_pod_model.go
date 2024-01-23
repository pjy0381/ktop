// save_pod_model.go
package model

import (
	"sort"

	"k8s.io/apimachinery/pkg/api/resource"
)

type SavedPodModel struct {
	Namespace string
	Name      string
	Status    string
	Node      string
	IP        string
	TimeSince string

	PodRequestedCpuQty *resource.Quantity
	PodRequestedMemQty *resource.Quantity
	PodUsageCpuQty     *resource.Quantity
	PodUsageMemQty     *resource.Quantity

	NodeAllocatableCpuQty *resource.Quantity
	NodeAllocatableMemQty *resource.Quantity
	NodeUsageCpuQty       *resource.Quantity
	NodeUsageMemQty       *resource.Quantity

	ReadyContainers int
	TotalContainers int
	Restarts        int
	Volumes         int
	VolMounts       int
}

// 복사 함수
func CopyPodModel(original *PodModel) *SavedPodModel {
	return &SavedPodModel{
		Namespace:             original.Namespace,
		Name:                  original.Name,
		Status:                original.Status,
		Node:                  original.Node,
		IP:                    original.IP,
		TimeSince:             original.TimeSince,
		PodRequestedCpuQty:    &resource.Quantity{}, // 빈 객체로 초기화
		PodRequestedMemQty:    &resource.Quantity{}, // 빈 객체로 초기화
		PodUsageCpuQty:        &resource.Quantity{}, // 빈 객체로 초기화
		PodUsageMemQty:        &resource.Quantity{}, // 빈 객체로 초기화
		NodeAllocatableCpuQty: &resource.Quantity{}, // 빈 객체로 초기화
		NodeAllocatableMemQty: &resource.Quantity{}, // 빈 객체로 초기화
		NodeUsageCpuQty:       &resource.Quantity{}, // 빈 객체로 초기화
		NodeUsageMemQty:       &resource.Quantity{}, // 빈 객체로 초기화
		ReadyContainers:       original.ReadyContainers,
		TotalContainers:       original.TotalContainers,
		Restarts:              original.Restarts,
		Volumes:               original.Volumes,
		VolMounts:             original.VolMounts,
	}
}

func SortSavedPodModels(savedPods []SavedPodModel) {
	sort.Slice(savedPods, func(i, j int) bool {
		if savedPods[i].Namespace != savedPods[j].Namespace {
			return savedPods[i].Namespace < savedPods[j].Namespace
		}
		return savedPods[i].Name < savedPods[j].Name
	})
}

