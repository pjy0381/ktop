package overview

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/pjy0381/ktop/application"
	"github.com/pjy0381/ktop/ui"
	"github.com/pjy0381/ktop/views/model"
	"k8s.io/apimachinery/pkg/api/resource"
)

type clusterSummaryPanel struct {
	app          *application.Application
	title        string
	root         *tview.Flex
	children     []tview.Primitive
	listCols     []string
	graphTable   *tview.Table
	summaryTable *tview.Table
}

func NewClusterSummaryPanel(app *application.Application, title string) ui.Panel {
	p := &clusterSummaryPanel{app: app, title: title}
	p.Layout(nil)
	p.children = append(p.children, p.graphTable)
	return p
}

func (p *clusterSummaryPanel) GetTitle() string {
	return p.title
}
func (p *clusterSummaryPanel) Layout(data interface{}) {
	p.summaryTable = tview.NewTable()
	p.summaryTable.SetBorder(false)
	p.summaryTable.SetBorders(false)
	p.summaryTable.SetTitleAlign(tview.AlignLeft)
	p.summaryTable.SetBorderColor(tcell.ColorWhite)

	p.graphTable = tview.NewTable()
	p.graphTable.SetBorder(false)
	p.graphTable.SetBorders(false)
	p.graphTable.SetTitleAlign(tview.AlignLeft)
	p.graphTable.SetBorderColor(tcell.ColorWhite)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.summaryTable, 1, 1, true).
		AddItem(p.graphTable, 1, 1, true)
	root.SetBorder(true)
	root.SetTitle(p.GetTitle())
	root.SetTitleAlign(tview.AlignLeft)
	root.SetBorderPadding(0, 0, 0, 0)

	p.root = root
}

func (p *clusterSummaryPanel) DrawHeader(data interface{}) {}

func (p *clusterSummaryPanel) DrawBody(data interface{}) {
	colorKeys := ui.ColorKeys{0: "green", 40: "yellow", 80: "red"}
	client := p.app.GetK8sClient()
	graphSize := 40
	switch summary := data.(type) {
	case model.ClusterSummary:
		var cpuRatio, memRatio ui.Ratio
		var cpuGraph, memGraph string
		var cpuMetrics, memMetrics string
		if err := client.AssertMetricsAvailable(); err != nil { // metrics not available
			cpuRatio = ui.GetRatio(float64(summary.RequestedPodCpuTotal.MilliValue()), float64(summary.AllocatableNodeCpuTotal.MilliValue()))
			cpuGraph = ui.BarGraph(graphSize, cpuRatio, colorKeys)
			cpuMetrics = fmt.Sprintf(
				"CPU: [white][%s[white]] %dm/%dm (%02.1f%% requested)",
				cpuGraph, summary.RequestedPodCpuTotal.MilliValue(), summary.AllocatableNodeCpuTotal.MilliValue(), cpuRatio*100,
			)

			memRatio = ui.GetRatio(float64(summary.RequestedPodMemTotal.MilliValue()), float64(summary.AllocatableNodeMemTotal.MilliValue()))
			memGraph = ui.BarGraph(graphSize, memRatio, colorKeys)
			memMetrics = fmt.Sprintf(
				"Memory: [white][%s[white]] %dGi/%dGi (%02.1f%% requested)",
				memGraph, summary.RequestedPodMemTotal.ScaledValue(resource.Giga), summary.AllocatableNodeMemTotal.ScaledValue(resource.Giga), memRatio*100,
			)
		} else {
			cpuRatio = ui.GetRatio(float64(summary.UsageNodeCpuTotal.MilliValue()), float64(summary.AllocatableNodeCpuTotal.MilliValue()))
			cpuGraph = ui.BarGraph(graphSize, cpuRatio, colorKeys)
			cpuMetrics = fmt.Sprintf(
				"CPU: [white][%s[white]] %dm/%dm (%02.1f%% used)",
				cpuGraph, summary.UsageNodeCpuTotal.MilliValue(), summary.AllocatableNodeCpuTotal.MilliValue(), cpuRatio*100,
			)

			memRatio = ui.GetRatio(float64(summary.UsageNodeMemTotal.MilliValue()), float64(summary.AllocatableNodeMemTotal.MilliValue()))
			memGraph = ui.BarGraph(graphSize, memRatio, colorKeys)
			memMetrics = fmt.Sprintf(
				"Memory: [white][%s[white]] %dGi/%dGi (%02.1f%% used)",
				memGraph, summary.UsageNodeMemTotal.ScaledValue(resource.Giga), summary.AllocatableNodeMemTotal.ScaledValue(resource.Giga), memRatio*100,
			)
		}

		p.graphTable.SetCell(
			0, 0,
			tview.NewTableCell(cpuMetrics).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft).
				SetExpansion(100),
		)

		p.graphTable.SetCell(
			0, 1,
			tview.NewTableCell(memMetrics).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft).
				SetExpansion(100),
		)

		// -=-=-=-=-=-=-=-=-=-=-=-=- cluster summary table -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-
		namespace := p.app.GetK8sClient().Namespace()
		if namespace == "" {
			namespace = "[Yellow](all)"
		}

		p.summaryTable.SetCell(
                        0, 0,
                        tview.NewTableCell(fmt.Sprintf("Selected Namespace: [white]%s", namespace)).
                                SetTextColor(tcell.ColorYellow).
                                SetAlign(tview.AlignLeft).
                                SetExpansion(100),
                )



		p.summaryTable.SetCell(
			0, 1,
			tview.NewTableCell(fmt.Sprintf("Nodes: " + getCountColor(summary.NodesReady, summary.NodesCount) + "%d[white]/%d", summary.NodesReady, summary.NodesCount)).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft).
				SetExpansion(100),
		)

		p.summaryTable.SetCell(
			0, 2,
			tview.NewTableCell(fmt.Sprintf("Pods: " + getCountColor(summary.PodsRunning, summary.PodsAvailable)  + "%d[white]/%d (%d imgs)", summary.PodsRunning, summary.PodsAvailable, summary.ImagesCount)).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft).
				SetExpansion(100),
		)

                p.summaryTable.SetCell(
                        0, 3,
                        tview.NewTableCell(fmt.Sprintf("Kubelet: " + getCountColor(summary.KubeletReady, summary.KubeletCount) + "%d[white]/%d", summary.KubeletReady, summary.KubeletCount)).
                                SetTextColor(tcell.ColorYellow).
                                SetAlign(tview.AlignLeft).
                                SetExpansion(100),
                )

		p.summaryTable.SetCell(
                        0, 4,
                        tview.NewTableCell(fmt.Sprintf("Containerd: " + getCountColor(summary.ContainerdReady, summary.ContainerdCount) + "%d[white]/%d", summary.ContainerdReady, summary.ContainerdCount)).
                                SetTextColor(tcell.ColorYellow).
                                SetAlign(tview.AlignLeft).
                                SetExpansion(100),
                )

		p.summaryTable.SetCell(
                        0, 5,
                        tview.NewTableCell(fmt.Sprintf("Scini: " + getCountColor(summary.SciniReady, summary.SciniCount)  + "%d[white]/%d", summary.SciniReady, summary.SciniCount)).
                                SetTextColor(tcell.ColorYellow).
                                SetAlign(tview.AlignLeft).
                                SetExpansion(100),
                )

                p.summaryTable.SetCell(
                        0, 6,
                        tview.NewTableCell(fmt.Sprintf("ETCD: " + getCountColor(summary.EtcdReady, summary.EtcdCount)  + "%d[white]/%d", summary.EtcdReady, summary.EtcdCount)).
                                SetTextColor(tcell.ColorYellow).
                                SetAlign(tview.AlignLeft).
                                SetExpansion(100),
                )

	default:
		panic(fmt.Sprintf("SummaryPanel.DrawBody: unexpected type %T", data))
	}
}

func getCountColor(ready, total int) string {
	if ready != total {
		return "[red]"
	}
	return "[green]"
}

func (p *clusterSummaryPanel) DrawFooter(data interface{}) {}

func (p *clusterSummaryPanel) Clear() {}

func (p *clusterSummaryPanel) GetRootView() tview.Primitive {
	return p.root
}

func (p *clusterSummaryPanel) GetChildrenViews() []tview.Primitive {
	return p.children
}

