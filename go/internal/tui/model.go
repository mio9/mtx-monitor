package tui

import (
	"context"
	"fmt"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"mio9/mtx-monitor/internal/bitrate"
	"mio9/mtx-monitor/internal/config"
	"mio9/mtx-monitor/internal/mediamtx"
	"mio9/mtx-monitor/internal/poll"
	"mio9/mtx-monitor/internal/watch"
)

// model is the Bubble Tea application model.
type model struct {
	state     DashboardState
	client    *mediamtx.Client
	tracker   *bitrate.Tracker
	quit      chan struct{}
	pollChan  chan poll.PollSnapshot
	errChan   chan error
	watchURLs []string
}

// NewModel creates a new Bubble Tea model with the given config.
func NewModel(cfg config.Config) (*model, error) {
	client := mediamtx.NewClient(cfg.APIURL, cfg.ApiAuth)
	tracker := bitrate.NewTracker()

	m := &model{
		state:    CreateInitialState(cfg),
		client:   client,
		tracker:  tracker,
		quit:     make(chan struct{}),
		pollChan: make(chan poll.PollSnapshot, 1),
		errChan:  make(chan error, 1),
	}

	return m, nil
}

// Init starts the polling goroutine.
func (m *model) Init() tea.Cmd {
	return m.startPolling()
}

// Update handles incoming messages.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg), nil
	case tea.WindowSizeMsg:
		return m, nil
	case poll.PollSnapshot:
		return m.applySnapshot(msg), nil
	case *pollError:
		m.state.Snapshot.PollError = msg.msg
		return m, nil
	case tea.QuitMsg:
		close(m.quit)
		return m, tea.Quit
	}
	return m, nil
}

// View renders the dashboard.
func (m *model) View() string {
	handlers := &tuiHandlers{model: m}
	return renderDashboard(m.state, handlers)
}

// handleKey routes key presses to actions.
func (m *model) handleKey(key tea.KeyMsg) tea.Model {
	if key.Type == tea.KeyCtrlC || key.String() == "q" {
		close(m.quit)
		return m
	}

	if m.state.ConfirmKick != nil {
		switch key.String() {
		case "esc", "n":
			m.state.ConfirmKick = nil
			msg := "kick cancelled"
			m.state.ActionMessage = &msg
		case "y":
			m.confirmKick()
		}
		return m
	}

	switch key.String() {
	case "left":
		m.state.ActiveSection = CycleDashboardSection(m.state.ActiveSection, -1)
	case "right":
		m.state.ActiveSection = CycleDashboardSection(m.state.ActiveSection, 1)
	case "w":
		if IsPublisherSection(m.state.ActiveSection) {
			selected := GetSelectedRow(m.state)
			if selected != nil {
				url, err := watch.WatchStream(selected.Name, m.state.Config.RTSPURL, m.state.Config.WatchPlayer)
				if err != nil {
					msg := "watch failed: " + err.Error()
					m.state.ActionMessage = &msg
				} else {
					m.watchURLs = append(m.watchURLs, url)
					msg := "watching " + selected.Name
					m.state.ActionMessage = &msg
				}
			}
		}
	case "k":
		if IsPublisherSection(m.state.ActiveSection) {
			selected := GetSelectedRow(m.state)
			if selected != nil {
				m.state.ConfirmKick = selected
			}
		}
	case "c":
		if m.state.ActiveSection == SectionViewers && m.state.ViewerUi.SelectedPathName != nil {
			m.state.ViewerUi = ToggleViewerPathCollapsed(m.state.ViewerUi, *m.state.ViewerUi.SelectedPathName)
		}
	case "p":
		if m.state.ActiveSection == SectionViewers && m.state.ViewerUi.SelectedPathName != nil {
			m.state.ViewerUi = ToggleViewerPathPinned(m.state.ViewerUi, *m.state.ViewerUi.SelectedPathName)
		}
	}

	return m
}

// applySnapshot updates state with a new poll result.
func (m *model) applySnapshot(snap poll.PollSnapshot) tea.Model {
	m.state.Snapshot = snap
	now := time.Now()
	ms := now.UnixMilli()
	m.state.LastUpdatedMs = &ms

	// Prune stale selections.
	m.state.SelectedKeys = PruneSelections(m.state.SelectedKeys, snap)

	// Prune stale viewer UI state.
	m.state.ViewerUi = PruneViewerUi(m.state.ViewerUi, snap.Viewers)

	// Clear action message after 3 seconds.
	if m.state.ActionMessage != nil {
		go func() {
			time.Sleep(3 * time.Second)
			select {
			case <-m.quit:
				return
			default:
			}
			m.state.ActionMessage = nil
		}()
	}

	return m
}

// confirmKick kicks the publisher and updates state.
func (m *model) confirmKick() {
	row := *m.state.ConfirmKick
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.client.KickPublisher(ctx, mediamtx.PathSource{Type: row.SourceType, ID: row.SourceID})
	if err != nil {
		m.state.Snapshot.Enforced = slices.DeleteFunc(m.state.Snapshot.Enforced, func(r poll.SessionRow) bool {
			return r.Name == row.Name
		})
		row.StatusDetail = err.Error()
		row.Status = poll.StatusKickFailed
		m.state.Snapshot.Enforced = append(m.state.Snapshot.Enforced, row)
		msg := "kick failed: " + err.Error()
		m.state.ActionMessage = &msg
	} else {
		m.tracker.Forget(row.Name)
		m.state.Snapshot.Enforced = slices.DeleteFunc(m.state.Snapshot.Enforced, func(r poll.SessionRow) bool {
			return r.Name == row.Name
		})
		row.Status = poll.StatusKicked
		row.StatusDetail = "kicked for exceeding limit"
		m.state.Snapshot.Enforced = append(m.state.Snapshot.Enforced, row)
		msg := "kicked " + row.Name
		m.state.ActionMessage = &msg
	}

	m.state.ConfirmKick = nil
}

// startPolling runs the background poll loop.
func (m *model) startPolling() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ticker := time.NewTicker(time.Duration(m.state.Config.PollIntervalMs) * time.Millisecond)
		defer ticker.Stop()

		opts := poll.RunPollCycleOptions{
			Client:           m.client,
			Tracker:          m.tracker,
			MaxBitrateBps:    m.state.Config.MaxBitrateBps,
			PathIncludeRegex: m.state.Config.PathIncludeRegex,
		}

		for {
			select {
			case <-m.quit:
				return nil
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}

			snap := poll.RunPollCycle(ctx, opts)
			if snap.PollError != "" {
				m.errChan <- &pollError{msg: snap.PollError}
			} else {
				select {
				case m.pollChan <- snap:
				default:
				}
			}
		}
	}
}

// Handlers defines callbacks from the view layer back into the model.
type Handlers interface {
	OnSectionChange(section DashboardSection)
	OnSelectRow(section DashboardSection, rowKey string)
	OnViewerFilterChange(query string)
	OnViewerSelectPath(pathName string)
	OnViewerToggleCollapse(pathName string)
	OnViewerTogglePin(pathName string)
	OnCancelKick()
	OnConfirmKick()
}

// tuiHandlers implements Handlers, routing view callbacks back to model.
type tuiHandlers struct {
	model *model
}

func (h *tuiHandlers) OnSectionChange(section DashboardSection) {
	h.model.state.ActiveSection = section
}

func (h *tuiHandlers) OnSelectRow(section DashboardSection, rowKey string) {
	h.model.state.SelectedKeys[string(section)] = rowKey
}

func (h *tuiHandlers) OnViewerFilterChange(query string) {
	h.model.state.ViewerUi.FilterQuery = query
}

func (h *tuiHandlers) OnViewerSelectPath(pathName string) {
	h.model.state.ViewerUi.SelectedPathName = &pathName
}

func (h *tuiHandlers) OnViewerToggleCollapse(pathName string) {
	h.model.state.ViewerUi = ToggleViewerPathCollapsed(h.model.state.ViewerUi, pathName)
}

func (h *tuiHandlers) OnViewerTogglePin(pathName string) {
	h.model.state.ViewerUi = ToggleViewerPathPinned(h.model.state.ViewerUi, pathName)
}

func (h *tuiHandlers) OnCancelKick() {
	h.model.state.ConfirmKick = nil
}

func (h *tuiHandlers) OnConfirmKick() {
	h.model.confirmKick()
}

// RunTUI starts the Bubble Tea TUI application with the given config.
func RunTUI(cfg config.Config) error {
	m, err := NewModel(cfg)
	if err != nil {
		return fmt.Errorf("create tui model: %w", err)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// pollError wraps a poll error message.
type pollError struct {
	msg string
}

func (e *pollError) Error() string { return e.msg }
