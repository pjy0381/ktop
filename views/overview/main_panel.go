package overview

import (
	"strconv"
	"strings"
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/pjy0381/ktop/application"
	"github.com/pjy0381/ktop/ui"
	"github.com/pjy0381/ktop/views/model"
)

type MainPanel struct {
        commandInput *tview.InputField
	app                 *application.Application
	title               string
	refresh             func()
	root                *tview.Flex
	children            []tview.Primitive
	selPanelIndex       int
	nodePanel           ui.Panel
	podPanel            ui.Panel
	clusterSummaryPanel ui.Panel

	nodePanelVisible    bool
	podPanelVisible     bool
	savePodPanelVisible bool
	savePodPanel	    ui.Panel
	lessPanel	    ui.Panel
	lessVisible	    bool

	sortPodBy	    int
	sortNodeBy	    int
	currentPodModels    []model.PodModel
	currentNpdeModels   []model.NodeModel

}

func New(app *application.Application, title string) *MainPanel {
	ctrl := &MainPanel{
		app:           app,
		title:         title,
		refresh:       app.Refresh,
		selPanelIndex: -1,
	}

	return ctrl
}

func (p *MainPanel) Layout(data interface{}) {
	p.initializeInputField()
	p.initializePanels()

	view := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.clusterSummaryPanel.GetRootView(), 4, 1, false).
		AddItem(p.commandInput, 1, 1, false)

	p.root = view
}

func (p *MainPanel) initializeInputField() {
	p.commandInput = tview.NewInputField()
	p.commandInput.SetInputCapture(p.handleInput)
	p.children = append(p.children, p.commandInput)
}

func (p *MainPanel) initializePanels() {
	p.nodePanel = NewNodePanel(p.app, fmt.Sprintf(" %c Nodes ", ui.Icons.Factory))
	p.nodePanel.DrawHeader([]string{"NAME", "STATUS", "AGE", "VERSION", "INT/EXT IPs", "OS/ARC", "PODS/IMGs", "DISK(allocatable)", "CPU", "MEM"})

	p.clusterSummaryPanel = NewClusterSummaryPanel(p.app, fmt.Sprintf(" %c Cluster Summary ", ui.Icons.Thermometer))
	p.clusterSummaryPanel.Layout(nil)
	p.clusterSummaryPanel.DrawHeader(nil)

	p.podPanel = NewPodPanel(p.app, fmt.Sprintf(" %c Pods ", ui.Icons.Package))
	p.podPanel.DrawHeader([]string{"NAMESPACE", "POD", "READY", "STATUS", "RESTARTS", "AGE", "VOLS", "IP", "NODE", "CPU", "MEMORY"})

	p.savePodPanel = NewPodPanel(p.app, fmt.Sprintf(" %c SavePods ", ui.Icons.Package))
        p.savePodPanel.DrawHeader([]string{"NAMESPACE", "POD", "READY", "STATUS", "RESTARTS", "AGE", "VOLS", "IP", "NODE", "CPU", "MEMORY"})
}

func CopyPodPanel(newPanel *podPanel) *podPanel {
    location, err := time.LoadLocation("Asia/Seoul")
    if err != nil {
		fmt.Println("타임존 로드 오류:", err)
		return nil
    }
    currentTime := time.Now().In(location)
    formattedTime := currentTime.Format("2006-01-02 15:04:05")

    copiedPanel := &podPanel{
        app:      newPanel.app,
        title:    fmt.Sprintf(" %c SavePods %s", ui.Icons.Package, formattedTime),
        root:     tview.NewFlex().SetDirection(tview.FlexRow),
        children: []tview.Primitive{},
        listCols: newPanel.listCols,
        list:     tview.NewTable(),
        laidout:  false,
    }
    copiedPanel.Layout(nil)

    // Copy the contents of the source list to the new list
    for row := 0; row < newPanel.list.GetRowCount(); row++ {
        for col := 0; col < newPanel.list.GetColumnCount(); col++ {
            cell := newPanel.list.GetCell(row, col)
            copiedPanel.list.SetCell(row, col, &tview.TableCell{
                Text:    cell.Text,
                Color:   cell.Color,
                Align:   cell.Align,
            })
        }
    }

    // Copy the header cells with proper SetExpansion
    for i, col := range copiedPanel.listCols {
        copiedPanel.list.SetCell(0, i,
            tview.NewTableCell(col).
                SetTextColor(tcell.ColorWhite).
                SetBackgroundColor(tcell.ColorDarkGreen).
                SetAlign(tview.AlignLeft).
                SetExpansion(1). // SetExpansion to 1 for each header cell
                SetSelectable(false),
        )
    }


    return copiedPanel
}

func LessPods(savePanel *podPanel, newPanel *podPanel) *podPanel {
    copiedPanel := &podPanel{
        app:      newPanel.app,
        title:    fmt.Sprintf(" %c LessPods", ui.Icons.Package),
        root:     tview.NewFlex().SetDirection(tview.FlexRow),
        children: []tview.Primitive{},
        listCols: newPanel.listCols,
        list:     tview.NewTable(),
        laidout:  false,
    }
    copiedPanel.Layout(nil)
    for row := 0; row < newPanel.list.GetRowCount(); row++ {
        found := false
        if newPanel == nil {
            break
        }
        if savePanel != nil {
            for col := 0; col < savePanel.list.GetRowCount(); col++ {
                if savePanel.list.GetCell(col, 1).Text == newPanel.list.GetCell(row, 1).Text {
                    found = true
                    break
                }
            }
        }
        if found {
            continue
        }
        for col := 0; col < newPanel.list.GetColumnCount(); col++ {
            cell := newPanel.list.GetCell(row, col)
            copiedPanel.list.SetCell(row, col, &tview.TableCell{
                Text: "[Green]" +  cell.Text,
                Color: cell.Color,
                Align: cell.Align,
            })
        }
    }

    for i, col := range copiedPanel.listCols {
        copiedPanel.list.SetCell(0, i,
            tview.NewTableCell(col).
                SetTextColor(tcell.ColorWhite).
                SetBackgroundColor(tcell.ColorDarkGreen).
                SetAlign(tview.AlignLeft).
                SetExpansion(1).
                SetSelectable(false),
        )
    }
    return copiedPanel
}

func (p *MainPanel) handleInput(event *tcell.EventKey) *tcell.EventKey {
    if event.Key() == tcell.KeyEnter {
        inputText := p.commandInput.GetText()
	commandText := strings.Split(inputText, " ")
        switch commandText[0] {
        case "q":
            p.app.Stop()
        case "n":
	    if len(commandText) == 1{
		p.togglePanel(&p.nodePanel, &p.nodePanelVisible)
	    }
        case "p":
	    // 정렬 기준 부여
	    sortBy := parseSortValue(commandText)
	    p.sortPodBy = sortBy

	    // 재정렬
	    model.SortPodModelsByField(p.currentPodModels, p.sortPodBy)
	    p.refreshPods(context.Background(), p.currentPodModels)

	    // view on/off
            if len(commandText) == 1 || !p.podPanelVisible {
		p.togglePanel(&p.podPanel, &p.podPanelVisible)
	    }
        case "s":
	    if(p.savePodPanelVisible){
		p.togglePanel(&p.savePodPanel, &p.savePodPanelVisible)
	    }
	    p.savePodPanel = CopyPodPanel(p.podPanel.(*podPanel))
	case "u":
            p.togglePanel(&p.savePodPanel, &p.savePodPanelVisible)
	case "v":
	    if !p.lessVisible {
		 p.lessPanel = LessPods(p.savePodPanel.(*podPanel), p.podPanel.(*podPanel))
	    }
	    p.togglePanel(&p.lessPanel, &p.lessVisible)
	case "c":
	    if p.nodePanelVisible {
		p.togglePanel(&p.nodePanel, &p.nodePanelVisible)
	    }
	    if p.podPanelVisible {
		p.togglePanel(&p.podPanel, &p.podPanelVisible)
	    }
            if p.savePodPanelVisible {
                p.togglePanel(&p.savePodPanel, &p.savePodPanelVisible)
            }
            if p.lessVisible {
                p.togglePanel(&p.lessPanel, &p.lessVisible)
            }
	}
        p.commandInput.SetText("")
    }
    return event
}

func parseSortValue(commandText []string) int {
    if len(commandText) > 1 {
        if sortBy, err := strconv.Atoi(commandText[1]); err == nil {
            return sortBy
        }
    }
    return 0
}


func (p *MainPanel) togglePanel(panel *ui.Panel, visible *bool) {
    if *visible {
        p.root.RemoveItem((*panel).GetRootView())
        p.children = removeChild(p.children, (*panel).GetRootView())
    } else {
        p.root.AddItem((*panel).GetRootView(), 0, 1, true)
        p.children = append(p.children, (*panel).GetRootView())
    }
    *visible = !*visible
}

func removeChild(children []tview.Primitive, target tview.Primitive) []tview.Primitive {
    for i, child := range children {
        if child == target {
            return append(children[:i], children[i+1:]...)
        }
    }
    return children
}

func (p *MainPanel) DrawHeader(_ interface{}) {}
func (p *MainPanel) DrawBody(_ interface{})   {}
func (p *MainPanel) DrawFooter(_ interface{}) {}
func (p *MainPanel) Clear()                   {}

func (p *MainPanel) GetTitle() string {
	return p.title
}
func (p *MainPanel) GetRootView() tview.Primitive {
	return p.root
}
func (p *MainPanel) GetChildrenViews() []tview.Primitive {
	return p.children
}

func (p *MainPanel) Run(ctx context.Context) error {
	p.Layout(nil)
	ctrl := p.app.GetK8sClient().Controller()
	ctrl.SetClusterSummaryRefreshFunc(p.refreshWorkloadSummary)
	ctrl.SetNodeRefreshFunc(p.refreshNodeView)
	ctrl.SetPodRefreshFunc(p.refreshPods)

	if err := ctrl.Start(ctx, time.Second*1); err != nil {
		panic(fmt.Sprintf("main panel: controller start: %s", err))
	}

	return nil
}

func (p *MainPanel) refreshNodeView(ctx context.Context, models []model.NodeModel) error {
	model.SortNodeModels(models)

	p.nodePanel.Clear()
	p.nodePanel.DrawBody(models)

	// required: always schedule screen refresh
	if p.refresh != nil {
		p.refresh()
	}

	return nil
}

func (p *MainPanel) refreshPods(ctx context.Context, models []model.PodModel) error {
	model.SortPodModelsByField(models, p.sortPodBy)
	p.currentPodModels = models

	// refresh pod list
	p.podPanel.Clear()
	p.podPanel.DrawBody(models)

	// required: always refresh screen
	if p.refresh != nil {
		p.refresh()
	}
	return nil
}

func (p *MainPanel) refreshWorkloadSummary(ctx context.Context, summary model.ClusterSummary) error {
	p.clusterSummaryPanel.Clear()
	p.clusterSummaryPanel.DrawBody(summary)
	if p.refresh != nil {
		p.refresh()
	}
	return nil
}
