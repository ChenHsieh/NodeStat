package ui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"nodestat/internal/models"
	"nodestat/internal/scheduler"
)

// App represents the main application model
type App struct {
	scheduler        scheduler.Scheduler
	currentPartition string
	partitions       []string
	nodes            []models.Node
	jobs             []models.Job
	userJobs         []models.Job
	currentUser      string
	nodesTable       table.Model
	stats            models.ClusterStats
	width            int
	height           int
	refreshInterval  time.Duration
	lastUpdate       time.Time
	keys             KeyMap
	err              error
}

// KeyMap defines the key bindings
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Quit     key.Binding
	Refresh  key.Binding
	Help     key.Binding
	Settings key.Binding
	Batch    key.Binding
	HighMem  key.Binding
	GPU      key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "move left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "move right"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r", " "),
			key.WithHelp("r/space", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Settings: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "settings"),
		),
		Batch: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "batch partition"),
		),
		HighMem: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "highmem partition"),
		),
		GPU: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "gpu partition"),
		),
	}
}

// NewApp creates a new application instance
func NewApp(schedulerType scheduler.SchedulerType, partition string) *App {
	s := scheduler.NewScheduler(schedulerType)

	// Get current user
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = "unknown"
	}

	app := &App{
		scheduler:        s,
		currentPartition: partition,
		currentUser:      currentUser,
		refreshInterval:  30 * time.Second,
		keys:             DefaultKeyMap(),
		partitions:       []string{"batch", "highmem_q", "gpu_q"},
	}

	// Initialize table
	app.initTable()

	return app
}

// initTable initializes the nodes table
func (a *App) initTable() {
	columns := []table.Column{
		{Title: "Node", Width: 10},
		{Title: "CPU", Width: 25},
		{Title: "Memory", Width: 25},
		{Title: "Avail CPU", Width: 8},
		{Title: "Avail Mem", Width: 8},
		{Title: "State", Width: 12},
		{Title: "Jobs", Width: 6},
	}

	a.nodesTable = table.New(
		table.WithColumns(columns),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	a.nodesTable.SetStyles(s)
}

// Styles for the application
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1)

	statsStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	progressBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205"))

	availableStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	usedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	offlineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.fetchData(),
		a.tick(),
	)
}

// tick returns a command for periodic updates
func (a *App) tick() tea.Cmd {
	return tea.Tick(a.refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Messages
type tickMsg time.Time
type dataMsg struct {
	nodes    []models.Node
	jobs     []models.Job
	userJobs []models.Job
	stats    models.ClusterStats
}
type errorMsg error

// fetchData fetches data from the scheduler
func (a *App) fetchData() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		nodes, err := a.scheduler.GetNodes(a.currentPartition)
		if err != nil {
			return errorMsg(err)
		}

		jobs, err := a.scheduler.GetJobs(a.currentPartition)
		if err != nil {
			return errorMsg(err)
		}

		userJobs, err := a.scheduler.GetUserJobs(a.currentUser)
		if err != nil {
			userJobs = []models.Job{} // Don't fail on user job error
		}

		// Sort nodes: IDLE nodes first, then by available resources (most powerful first)
		sort.Slice(nodes, func(i, j int) bool {
			ni, nj := &nodes[i], &nodes[j]

			// Prioritize available nodes
			if ni.IsAvailable() != nj.IsAvailable() {
				return ni.IsAvailable()
			}

			// Among available nodes, sort by total available power (CPU + memory)
			if ni.IsAvailable() && nj.IsAvailable() {
				iPower := ni.GetAvailCores()*1000 + ni.GetAvailMemGB()
				jPower := nj.GetAvailCores()*1000 + nj.GetAvailMemGB()
				return iPower > jPower
			}

			// Among unavailable nodes, sort by state priority
			stateOrder := map[models.NodeState]int{
				models.StateRunning: 0,
				models.StateBusy:    1,
				models.StateDrained: 2,
				models.StateDown:    3,
				models.StateOffline: 4,
			}

			return stateOrder[ni.State] < stateOrder[nj.State]
		})

		stats := a.calculateStats(nodes)

		return dataMsg{
			nodes:    nodes,
			jobs:     jobs,
			userJobs: userJobs,
			stats:    stats,
		}
	})
}

// calculateStats calculates cluster statistics
func (a *App) calculateStats(nodes []models.Node) models.ClusterStats {
	stats := models.ClusterStats{}

	for _, node := range nodes {
		stats.TotalNodes++
		stats.TotalCores += node.TotalCores
		stats.UsedCores += node.UsedCores
		stats.TotalMemoryGB += node.GetTotalMemGB()
		stats.UsedMemoryGB += node.GetUsedMemGB()

		if node.IsAvailable() {
			stats.AvailNodes++
		}
	}

	stats.AvailCores = stats.TotalCores - stats.UsedCores
	stats.AvailMemoryGB = stats.TotalMemoryGB - stats.UsedMemoryGB

	return stats
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.nodesTable.SetWidth(msg.Width - 4)
		// Calculate table height: total height - title(1) - header(1) - stats(4) - jobs(1) - help(1) - margins(4)
		tableHeight := msg.Height - 12
		if tableHeight < 5 {
			tableHeight = 5
		}
		a.nodesTable.SetHeight(tableHeight)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, a.keys.Quit):
			return a, tea.Quit

		case key.Matches(msg, a.keys.Refresh):
			cmds = append(cmds, a.fetchData())

		case key.Matches(msg, a.keys.Batch):
			a.currentPartition = "batch"
			cmds = append(cmds, a.fetchData())

		case key.Matches(msg, a.keys.HighMem):
			a.currentPartition = "highmem_q"
			cmds = append(cmds, a.fetchData())

		case key.Matches(msg, a.keys.GPU):
			a.currentPartition = "gpu_q"
			cmds = append(cmds, a.fetchData())
		}

	case tickMsg:
		cmds = append(cmds, a.tick(), a.fetchData())

	case dataMsg:
		a.nodes = msg.nodes
		a.jobs = msg.jobs
		a.userJobs = msg.userJobs
		a.stats = msg.stats
		a.lastUpdate = time.Now()
		a.err = nil
		a.updateTable()

	case errorMsg:
		a.err = msg
	}

	// Update table navigation
	var cmd tea.Cmd
	a.nodesTable, cmd = a.nodesTable.Update(msg)
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

// updateTable updates the table with current node data
func (a *App) updateTable() {
	var rows []table.Row

	for _, node := range a.nodes {
		// Check if user has jobs on this node
		userHasJobs := a.userHasJobsOnNode(node.ID)

		rows = append(rows, table.Row{
			a.formatNodeID(node.ID, userHasJobs),
			a.formatResourceBar(node.UsedCores, node.TotalCores, "CPU"),
			a.formatResourceBar(node.GetUsedMemGB(), node.GetTotalMemGB(), "MEM"),
			fmt.Sprintf("%d", node.GetAvailCores()),
			fmt.Sprintf("%d GB", node.GetAvailMemGB()),
			a.formatNodeState(node.State),
			fmt.Sprintf("%d", len(node.Jobs)),
		})
	}

	a.nodesTable.SetRows(rows)
}

// userHasJobsOnNode checks if the current user has jobs on the given node
func (a *App) userHasJobsOnNode(nodeID string) bool {
	for _, job := range a.userJobs {
		if job.State == models.JobRunning {
			for _, jobNode := range job.NodeList {
				if jobNode == nodeID {
					return true
				}
			}
		}
	}
	return false
}

// formatNodeID formats the node ID with highlighting for user jobs
func (a *App) formatNodeID(nodeID string, userHasJobs bool) string {
	if userHasJobs {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true).
			Render("★ " + nodeID)
	}
	return nodeID
}

// formatResourceBar creates a visual progress bar for resource usage
func (a *App) formatResourceBar(used, total int, resourceType string) string {
	if total == 0 {
		return strings.Repeat("░", 20) + " 0/0"
	}

	percentage := float64(used) / float64(total)
	barLength := 20
	filledLength := int(percentage * float64(barLength))

	var bar strings.Builder

	// Used portion (red)
	bar.WriteString(usedStyle.Render(strings.Repeat("█", filledLength)))
	// Available portion (green)
	bar.WriteString(availableStyle.Render(strings.Repeat("░", barLength-filledLength)))

	// Add text info
	bar.WriteString(fmt.Sprintf(" %d/%d", used, total))

	return bar.String()
}

// formatNodeState formats the node state with appropriate styling
func (a *App) formatNodeState(state models.NodeState) string {
	switch state {
	case models.StateIdle:
		return availableStyle.Render(string(state))
	case models.StateRunning:
		return progressBarStyle.Render(string(state))
	case models.StateDown, models.StateOffline:
		return offlineStyle.Render(string(state))
	default:
		return usedStyle.Render(string(state))
	}
}

// View implements tea.Model
func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	var sections []string

	// Title with margin
	sections = append(sections, titleStyle.Render("🖥️  NodeStat - Cluster Monitor"))
	sections = append(sections, "") // Add spacing

	// Error display
	if a.err != nil {
		sections = append(sections, errorStyle.Render(fmt.Sprintf("Error: %v", a.err)))
		sections = append(sections, "") // Add spacing after error
	}

	// Current partition and stats
	sections = append(sections, a.renderHeader())
	sections = append(sections, a.renderStats())
	sections = append(sections, "") // Add spacing

	// Nodes table
	sections = append(sections, headerStyle.Render("Nodes"))
	sections = append(sections, a.nodesTable.View())

	// Jobs summary and help
	sections = append(sections, "")
	sections = append(sections, a.renderJobsSummary())
	sections = append(sections, a.renderHelp())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the current partition and last update info
func (a *App) renderHeader() string {
	partition := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true).
		Render(fmt.Sprintf("Partition: %s", a.currentPartition))

	lastUpdate := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("Last update: %s", a.lastUpdate.Format("15:04:05")))

	return lipgloss.JoinHorizontal(lipgloss.Top, partition, "  ", lastUpdate)
}

// renderStats renders cluster statistics
func (a *App) renderStats() string {
	cpuBar := a.formatResourceBar(a.stats.UsedCores, a.stats.TotalCores, "CPU")
	memBar := a.formatResourceBar(a.stats.UsedMemoryGB, a.stats.TotalMemoryGB, "MEM")

	nodeStats := fmt.Sprintf("Nodes: %d total, %d available",
		a.stats.TotalNodes, a.stats.AvailNodes)

	return statsStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			fmt.Sprintf("CPU  %s", cpuBar),
			fmt.Sprintf("MEM  %s", memBar),
			nodeStats,
		),
	)
}

// renderJobsSummary renders a summary of jobs
func (a *App) renderJobsSummary() string {
	runningJobs := len(a.jobs)
	userJobs := len(a.userJobs)

	return headerStyle.Render(fmt.Sprintf("Jobs: %d running (%d yours)", runningJobs, userJobs))
}

// renderHelp renders help information
func (a *App) renderHelp() string {
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("b: batch | m: highmem | g: gpu | r: refresh | q: quit")

	return help
}

// Run starts the application
func (a *App) Run() error {
	p := tea.NewProgram(a, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
