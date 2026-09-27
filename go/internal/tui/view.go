package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"mio9/mtx-monitor/internal/bitrate"
	"mio9/mtx-monitor/internal/poll"
)

const VIEWER_SESSION_LIST_MAX_HEIGHT = 8

var (
	headerStyle           = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	tabActiveStyle        = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("170")).Padding(0, 1).Bold(true)
	tabInactiveStyle      = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	panelStyle            = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1).Width(0)
	footerStyle           = lipgloss.NewStyle().Padding(1, 0).Foreground(lipgloss.Color("243"))
	statusBarStyle        = lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("244"))
	actionMsgStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).BorderForeground(lipgloss.Color("39")).Foreground(lipgloss.Color("39"))
	errorMsgStyle         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).BorderForeground(lipgloss.Color("167")).Foreground(lipgloss.Color("167"))
	dialogBgStyle         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).Background(lipgloss.Color("235")).Foreground(lipgloss.Color("243")).Width(50)
	dialogBtnStyle        = lipgloss.NewStyle().Padding(0, 2).Background(lipgloss.Color("237")).Foreground(lipgloss.Color("255"))
	dialogBtnConfirmStyle = dialogBtnStyle.Background(lipgloss.Color("166")).Foreground(lipgloss.Color("255"))
	dialogOverlayStyle    = lipgloss.NewStyle().Background(lipgloss.Color("0"))
)

// renderDashboard renders the full dashboard.
func renderDashboard(state DashboardState, handlers Handlers) string {
	if state.ConfirmKick != nil {
		return renderWithDialog(state, handlers)
	}
	return renderMain(state, handlers)
}

func renderMain(state DashboardState, handlers Handlers) string {
	var updatedStr string
	if state.LastUpdatedMs != nil {
		updatedStr = formatTimestamp(*state.LastUpdatedMs)
	} else {
		updatedStr = "never"
	}

	// Status bar.
	statusLeft := lipgloss.JoinHorizontal(lipgloss.Bottom,
		lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Render("mtx-watcher"),
	)
	statusRight := lipgloss.JoinHorizontal(lipgloss.Bottom,
		"updated "+updatedStr,
		" | ",
		getStatusHints(state.ActiveSection),
	)
	statusBar := lipgloss.NewStyle().Width(80).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, statusLeft, lipgloss.NewStyle().Width(50).Render(statusRight)),
	)

	// Action/error messages.
	var msgLines []string
	if state.ActionMessage != nil {
		msgLines = append(msgLines, actionMsgStyle.Render(*state.ActionMessage))
	}
	if state.Snapshot.PollError != "" {
		msgLines = append(msgLines, errorMsgStyle.Render(state.Snapshot.PollError))
	}

	// Tabs.
	enforcedTab := sectionTab("enforced", "Enforced", len(state.Snapshot.Enforced), state.ActiveSection)
	otherTab := sectionTab("other", "Other publishers", len(state.Snapshot.Other), state.ActiveSection)
	viewerCount := len(FlattenViewers(state.Snapshot))
	viewersTab := sectionTab("viewers", "Viewers", viewerCount, state.ActiveSection)

	tabs := lipgloss.NewStyle().Width(80).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, enforcedTab, otherTab, viewersTab),
	)

	selectedLabel := ""
	if IsPublisherSection(state.ActiveSection) {
		row := GetSelectedRow(state)
		if row != nil {
			selectedLabel = row.Name
		} else {
			selectedLabel = "none"
		}
	} else {
		if state.ViewerUi.SelectedPathName != nil {
			selectedLabel = *state.ViewerUi.SelectedPathName
		} else {
			selectedLabel = "none"
		}
	}

	// Main content panel.
	var content string
	switch state.ActiveSection {
	case SectionEnforced:
		content = sessionTable("enforced", SectionEnforced, state.Snapshot.Enforced, state.Config.MaxBitrateBps, true, state.SelectedKeys["enforced"], state.ActiveSection, handlers)
	case SectionOther:
		content = sessionTable("other", SectionOther, state.Snapshot.Other, state.Config.MaxBitrateBps, false, state.SelectedKeys["other"], state.ActiveSection, handlers)
	case SectionViewers:
		content = viewersPanel(state, handlers)
	}

	mainContent := lipgloss.NewStyle().Padding(1).Width(80).Render(
		lipgloss.JoinVertical(lipgloss.Top,
			statusBar,
			strings.Join(msgLines, "\n"),
			tabs,
			"selected: "+selectedLabel,
			panelStyle.Render(content),
		),
	)

	return mainContent
}

func renderWithDialog(state DashboardState, handlers Handlers) string {
	main := renderMainNoWidth(state, handlers)
	dialog := kickConfirmDialog(*state.ConfirmKick, handlers)
	return dialogOverlayStyle.Width(80).Height(24).Render(
		lipgloss.JoinVertical(lipgloss.Center,
			main,
			dialogBgStyle.Render(dialog),
		),
	)
}

func renderMainNoWidth(state DashboardState, handlers Handlers) string {
	var msgLines []string
	if state.ActionMessage != nil {
		msgLines = append(msgLines, actionMsgStyle.Render(*state.ActionMessage))
	}
	if state.Snapshot.PollError != "" {
		msgLines = append(msgLines, errorMsgStyle.Render(state.Snapshot.PollError))
	}

	enforcedTab := sectionTab(string(SectionEnforced), "Enforced", len(state.Snapshot.Enforced), state.ActiveSection)
	otherTab := sectionTab(string(SectionOther), "Other publishers", len(state.Snapshot.Other), state.ActiveSection)
	viewerCount := len(FlattenViewers(state.Snapshot))
	viewersTab := sectionTab(string(SectionViewers), "Viewers", viewerCount, state.ActiveSection)

	tabs := lipgloss.JoinHorizontal(lipgloss.Top, enforcedTab, otherTab, viewersTab)

	var content string
	switch state.ActiveSection {
	case SectionEnforced:
		content = sessionTable("enforced", SectionEnforced, state.Snapshot.Enforced, state.Config.MaxBitrateBps, true, state.SelectedKeys["enforced"], state.ActiveSection, handlers)
	case SectionOther:
		content = sessionTable("other", SectionOther, state.Snapshot.Other, state.Config.MaxBitrateBps, false, state.SelectedKeys["other"], state.ActiveSection, handlers)
	case SectionViewers:
		content = viewersPanel(state, handlers)
	}

	return lipgloss.JoinVertical(lipgloss.Top,
		tabs,
		content,
	)
}

func getStatusHints(activeSection DashboardSection) string {
	if activeSection == SectionViewers {
		return "↑/↓ path | c collapse | p pin | Tab filter | q quit"
	}
	return "←/→ section | w watch | k kick | q quit"
}

func formatTimestamp(timeMs int64) string {
	return time.UnixMilli(timeMs).Format("15:04:05")
}

func statusBadge(row poll.SessionRow, enforced bool) string {
	var color lipgloss.Color
	switch row.Status {
	case "warming":
		color = lipgloss.Color("39") // info blue
	case "ok":
		color = lipgloss.Color("42") // success green
	case "over":
		if enforced {
			color = lipgloss.Color("167") // error red
		} else {
			color = lipgloss.Color("214") // warning yellow
		}
	case "kicked", "kick_failed":
		color = lipgloss.Color("167") // error red
	default:
		color = lipgloss.Color("243") // dim gray
	}

	label := string(row.Status)
	if row.Status == "over" && !enforced {
		label = "over (not enforced)"
	}
	return lipgloss.NewStyle().Padding(0, 1).Background(color).Foreground(lipgloss.Color("235")).Render(label)
}

func sectionTab(section string, label string, count int, activeSection DashboardSection) string {
	active := section == string(activeSection)
	if active {
		return tabActiveStyle.Render("▶ " + label + " (" + fmt.Sprintf("%d", count) + ")")
	}
	return tabInactiveStyle.Render("  " + label + " (" + fmt.Sprintf("%d", count) + ")")
}

func sessionTable(_ string, section DashboardSection, rows []poll.SessionRow, maxBps int, _ bool, _ string, activeSection DashboardSection, _ Handlers) string {
	limitLabel := bitrate.FormatBitrate(maxBps)
	isActive := section == activeSection

	if len(rows) == 0 {
		return lipgloss.NewStyle().Padding(1).Foreground(lipgloss.Color("243")).Render("No active publishing paths")
	}

	columns := []table.Column{
		{Title: "Path", Width: 25},
		{Title: "Protocol", Width: 14},
		{Title: "Bitrate", Width: 14},
		{Title: "Limit", Width: 14},
		{Title: "Status", Width: 22},
	}

	type rowData struct {
		name       string
		sourceType string
		bitrateBps *int
		limit      string
		status     string
		enforced   bool
	}

	data := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		bitrateStr := "—"
		if row.BitrateBps != nil {
			bitrateStr = bitrate.FormatBitrate(*row.BitrateBps)
		}
		data = append(data, table.Row{
			row.Name,
			row.SourceType,
			bitrateStr,
			limitLabel,
			string(row.Status),
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(data),
		table.WithFocused(isActive),
		table.WithHeight(len(rows)+1),
	)

	styles := table.DefaultStyles()
	styles.Header = lipgloss.NewStyle().Bold(true).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	styles.Selected = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("235")).Padding(0, 1)
	t.SetStyles(styles)

	return t.View()
}

func encodePathId(pathName string) string {
	var b strings.Builder
	for _, c := range pathName {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			b.WriteRune(c)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func viewerSessionListHeight(count int) int {
	if count == 0 {
		return 0
	}
	if count > VIEWER_SESSION_LIST_MAX_HEIGHT {
		return VIEWER_SESSION_LIST_MAX_HEIGHT
	}
	return count
}

func estimateViewerGroupHeight(group poll.ViewerGroup, collapsed bool) int {
	if collapsed {
		return 2
	}
	if len(group.Readers) == 0 {
		return 4
	}
	return 5 + viewerSessionListHeight(len(group.Readers))
}

func formatViewerReaderLine(reader poll.ViewerRow) string {
	outboundLabel := "—"
	if reader.OutboundBytes != nil {
		outboundLabel = bitrate.FormatBytes(*reader.OutboundBytes)
	}
	return fmt.Sprintf("%-14s%-10s%-16s%s", reader.ReaderType, reader.SessionPrefix, reader.RemoteAddr, outboundLabel)
}

func viewerGroupHeader(group poll.ViewerGroup, collapsed, pinned, focused bool) string {
	markers := ""
	if pinned {
		markers += "*"
	} else {
		markers += " "
	}
	if collapsed {
		markers += "[+]"
	} else {
		markers += "[-]"
	}
	if focused {
		markers += ">"
	} else {
		markers += " "
	}

	outboundLabel := "est —"
	if group.EstimatedOutboundBps != nil {
		outboundLabel = "est " + bitrate.FormatBitrate(*group.EstimatedOutboundBps)
	}
	publisherLabel := "src —"
	if group.PublisherBitrateBps != nil {
		publisherLabel = "src " + bitrate.FormatBitrate(*group.PublisherBitrateBps)
	}

	label := fmt.Sprintf("%s %s (%d viewer%s, %s, %s)",
		markers, group.PathName, len(group.Readers),
		func() string {
			if len(group.Readers) == 1 {
				return ""
			}
			return "s"
		}(),
		publisherLabel, outboundLabel)

	style := lipgloss.NewStyle()
	if focused || pinned {
		style = style.Bold(true)
	}
	return style.Render(label)
}

func viewerSessionList(group poll.ViewerGroup) string {
	sessionCount := len(group.Readers)
	if sessionCount == 0 {
		return "  no sessions"
	}

	needsScroll := sessionCount > VIEWER_SESSION_LIST_MAX_HEIGHT
	var countLabel string
	if needsScroll {
		countLabel = fmt.Sprintf("  %d sessions (scroll for more)", sessionCount)
	} else {
		suffix := "s"
		if sessionCount == 1 {
			suffix = ""
		}
		countLabel = fmt.Sprintf("  %d session%s", sessionCount, suffix)
	}

	lines := []string{countLabel}
	lines = append(lines, "  Protocol        Session  Remote              Outbound")

	for _, r := range group.Readers {
		lines = append(lines, "  "+formatViewerReaderLine(r))
	}

	return strings.Join(lines, "\n")
}

func viewersPanel(state DashboardState, _ Handlers) string {
	preparedGroups := PrepareViewerGroups(state.Snapshot.Viewers, state.ViewerUi)
	totalPathCount := len(state.Snapshot.Viewers)
	totalSessionCount := len(FlattenViewers(state.Snapshot))

	visibleSessionCount := 0
	for _, g := range preparedGroups {
		visibleSessionCount += len(g.Readers)
	}

	sessionCountLabel := fmt.Sprintf("%d sessions", totalSessionCount)
	if visibleSessionCount != totalSessionCount {
		sessionCountLabel = fmt.Sprintf("%d/%d sessions", visibleSessionCount, totalSessionCount)
	}

	serverOutboundLabel := "est server outbound —"
	if state.Snapshot.EstimatedServerOutboundBps != nil {
		serverOutboundLabel = "est server outbound " + bitrate.FormatBitrate(*state.Snapshot.EstimatedServerOutboundBps)
	}

	if totalPathCount == 0 {
		return lipgloss.NewStyle().Padding(1).Foreground(lipgloss.Color("243")).Render("No connected viewers")
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Padding(0, 1).Width(76).Render(
		lipgloss.JoinVertical(lipgloss.Top,
			fmt.Sprintf("Filter paths... (query: %q)", state.ViewerUi.FilterQuery),
			fmt.Sprintf("%s | %d/%d paths | %s | * pin | c collapse | p pin",
				sessionCountLabel, len(preparedGroups), totalPathCount, serverOutboundLabel),
		),
	))

	if len(preparedGroups) == 0 {
		lines = append(lines, "No paths match filter")
	} else {
		for _, g := range preparedGroups {
			collapsed := IsViewerPathCollapsed(state.ViewerUi, g.PathName)
			pinned := IsViewerPathPinned(state.ViewerUi, g.PathName)
			focused := state.ViewerUi.SelectedPathName != nil && *state.ViewerUi.SelectedPathName == g.PathName

			grpLines := []string{viewerGroupHeader(g, collapsed, pinned, focused)}
			if !collapsed {
				grpLines = append(grpLines, viewerSessionList(g))
			}
			lines = append(lines, panelStyle.Width(76).Render(strings.Join(grpLines, "\n")))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Top, lines...)
}

func kickConfirmDialog(row poll.SessionRow, _ Handlers) string {
	msg := fmt.Sprintf("Kick \"%s\" (%s)?", row.Name, row.SourceType)
	cancelBtn := dialogBtnStyle.Render("[Cancel]")
	confirmBtn := dialogBtnConfirmStyle.Render("[Kick]")
	return lipgloss.JoinVertical(lipgloss.Top,
		lipgloss.NewStyle().Bold(true).Render("Kick publisher?"),
		msg,
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, cancelBtn, confirmBtn),
	)
}
