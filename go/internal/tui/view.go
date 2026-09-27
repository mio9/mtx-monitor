package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"mio9/mtx-monitor/internal/bitrate"
	"mio9/mtx-monitor/internal/constants"
	"mio9/mtx-monitor/internal/poll"
)

const (
	VIEWER_SESSION_LIST_MAX_HEIGHT = 8
	frameGutter                    = 1
	panelPadding                   = 1
	fallbackFrameWidth             = 72
)

var (
	headerStyle           = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	tabActiveStyle        = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("170")).Padding(0, 1).Bold(true).MarginRight(1)
	tabInactiveStyle      = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1).MarginRight(1)
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
func renderDashboard(state DashboardState, handlers Handlers, width int, height int) string {
	frame := usableWidth(width)
	if state.ConfirmKick != nil {
		return renderWithDialog(state, handlers, frame, height)
	}
	return renderMain(state, handlers, frame)
}

func renderMain(state DashboardState, handlers Handlers, frame int) string {
	inner := frame - frameGutter*2
	panelBoxWidth := inner - 2
	textWidth := panelBoxWidth - panelPadding*2
	if textWidth < 20 {
		textWidth = 20
		panelBoxWidth = textWidth + panelPadding*2
		inner = panelBoxWidth + 2
	}

	var updatedStr string
	if state.LastUpdatedMs != nil {
		updatedStr = formatTimestamp(*state.LastUpdatedMs)
	} else {
		updatedStr = "never"
	}

	var msgLines []string
	if state.ActionMessage != nil {
		msgLines = append(msgLines, actionMsgStyle.Render(*state.ActionMessage))
	}
	if state.Snapshot.PollError != "" {
		msgLines = append(msgLines, errorMsgStyle.Render(clipTo(state.Snapshot.PollError, inner-4)))
	}

	selectedLabel := ""
	if IsPublisherSection(state.ActiveSection) {
		row := GetSelectedRow(state)
		if row != nil {
			selectedLabel = row.Name
		} else {
			selectedLabel = "none"
		}
	} else if state.ActiveSection == SectionViewers {
		if state.ViewerUi.SelectedPathName != nil {
			selectedLabel = *state.ViewerUi.SelectedPathName
		} else {
			selectedLabel = "none"
		}
	}

	var content string
	switch state.ActiveSection {
	case SectionEnforced:
		content = sessionTable(state.Snapshot.Enforced, state.Config.MaxBitrateBps, true, state.SelectedKeys["enforced"], textWidth)
	case SectionOther:
		content = sessionTable(state.Snapshot.Other, state.Config.MaxBitrateBps, false, state.SelectedKeys["other"], textWidth)
	case SectionViewers:
		content = viewersPanel(state, handlers, textWidth)
	case SectionInstance:
		content = instancePanel(state, textWidth)
	}

	body := []string{
		statusBar(inner, updatedStr, getStatusHints(state.ActiveSection)),
		"",
	}
	if len(msgLines) > 0 {
		body = append(body, strings.Join(msgLines, "\n"), "")
	}
	body = append(body, sectionTabs(state, inner), "")
	if state.ActiveSection != SectionInstance {
		body = append(body, "selected  "+clipTo(selectedLabel, inner-10), "")
	}
	body = append(body, panelStyle.Width(panelBoxWidth).Render(content))

	return lipgloss.NewStyle().Padding(frameGutter, frameGutter).Render(
		lipgloss.JoinVertical(lipgloss.Top, body...),
	)
}

func renderWithDialog(state DashboardState, handlers Handlers, frame int, height int) string {
	main := renderMain(state, handlers, frame)
	dialog := dialogBgStyle.Render(kickConfirmDialog(*state.ConfirmKick, handlers))
	if height < 8 {
		height = 24
	}
	return lipgloss.Place(frame, height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, main, "", dialog),
	)
}

func usableWidth(width int) int {
	if width <= 0 {
		return fallbackFrameWidth
	}
	return width
}

func statusBar(inner int, updated string, hints string) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).Render(constants.CommandName)
	right := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("updated  " + updated)
	gap := inner - lipgloss.Width(title) - lipgloss.Width(right)
	var titleLine string
	if gap >= 2 {
		titleLine = title + strings.Repeat(" ", gap) + right
	} else {
		titleLine = title + "\n" + right
	}
	hintLine := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(hints)
	if lipgloss.Width(hintLine) > inner {
		hintLine = clipTo(hints, inner)
	}
	return titleLine + "\n" + hintLine
}

func sectionTabs(state DashboardState, inner int) string {
	enforcedTab := sectionTab("enforced", "Enforced", len(state.Snapshot.Enforced), state.ActiveSection, true)
	otherTab := sectionTab("other", "Other", len(state.Snapshot.Other), state.ActiveSection, true)
	viewerCount := len(FlattenViewers(state.Snapshot))
	viewersTab := sectionTab("viewers", "Viewers", viewerCount, state.ActiveSection, true)
	instanceTab := sectionTab("instance", "Instance", 0, state.ActiveSection, false)
	tabs := lipgloss.JoinHorizontal(lipgloss.Top, enforcedTab, otherTab, viewersTab, instanceTab)
	if lipgloss.Width(tabs) <= inner {
		return tabs
	}

	return compactTabs(state)
}

func compactTabs(state DashboardState) string {
	parts := []struct {
		section DashboardSection
		label   string
		count   int
		show    bool
	}{
		{SectionEnforced, "Enforced", len(state.Snapshot.Enforced), true},
		{SectionOther, "Other", len(state.Snapshot.Other), true},
		{SectionViewers, "Viewers", len(FlattenViewers(state.Snapshot)), true},
		{SectionInstance, "Instance", 0, false},
	}
	var rendered []string
	for _, part := range parts {
		text := part.label
		if part.show {
			text = fmt.Sprintf("%s (%d)", part.label, part.count)
		}
		style := lipgloss.NewStyle().Padding(0, 1).MarginRight(1)
		if part.section == state.ActiveSection {
			style = style.Bold(true).Underline(true).Foreground(lipgloss.Color("170"))
		} else {
			style = style.Foreground(lipgloss.Color("245"))
		}
		rendered = append(rendered, style.Render(text))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func getStatusHints(activeSection DashboardSection) string {
	if activeSection == SectionViewers {
		return "↑/↓ path | c collapse | p pin | Tab filter | q quit"
	}
	if activeSection == SectionInstance {
		return "←/→ section | q quit"
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

func sectionTab(section string, label string, count int, activeSection DashboardSection, showCount bool) string {
	text := label
	if showCount {
		text = label + " (" + fmt.Sprintf("%d", count) + ")"
	}
	active := section == string(activeSection)
	if active {
		return tabActiveStyle.Render("▶ " + text)
	}
	return tabInactiveStyle.Render("  " + text)
}

func instancePanel(state DashboardState, textWidth int) string {
	auth := "none"
	if state.Config.ApiAuth != nil {
		if state.Config.ApiAuth.Scheme == "bearer" {
			auth = "bearer"
		} else {
			auth = "basic user=" + state.Config.ApiAuth.Username
		}
	}

	pathFilter := "all"
	if state.Config.PathIncludeRegex != nil {
		pathFilter = "/" + state.Config.PathIncludeRegex.String() + "/"
	}

	version := "—"
	started := "—"
	connection := "connecting"
	if state.Snapshot.Server.Error != "" {
		connection = "unreachable"
	} else if state.Snapshot.Server.Version != "" {
		version = state.Snapshot.Server.Version
		started = formatServerStarted(state.Snapshot.Server.Started)
		connection = "connected"
	}

	lines := []string{
		fmt.Sprintf("%-14s%s", "API", state.Config.APIURL),
		fmt.Sprintf("%-14s%s", "RTSP", state.Config.RTSPURL),
		fmt.Sprintf("%-14s%s", "Auth", auth),
		fmt.Sprintf("%-14s%s", "Version", version),
		fmt.Sprintf("%-14s%s", "Started", started),
		fmt.Sprintf("%-14s%dms", "Poll", state.Config.PollIntervalMs),
		fmt.Sprintf("%-14s%s", "Limit", bitrate.FormatBitrate(state.Config.MaxBitrateBps)),
		fmt.Sprintf("%-14s%s", "Path filter", pathFilter),
		fmt.Sprintf("%-14s%s", "Status", connection),
	}
	if state.Snapshot.Server.Error != "" {
		lines = append(lines, "", state.Snapshot.Server.Error)
	}
	for i, line := range lines {
		lines[i] = clipTo(line, textWidth)
	}
	return strings.Join(lines, "\n")
}

func formatServerStarted(started string) string {
	if started == "" {
		return "—"
	}
	parsed, err := time.Parse(time.RFC3339, started)
	if err != nil {
		return started
	}
	return parsed.Local().Format("2006-01-02 15:04:05")
}

func sessionTable(rows []poll.SessionRow, maxBps int, enforced bool, selectedName string, textWidth int) string {
	if len(rows) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render("No active publishing paths")
	}

	pathW, protocolW, bitrateW, limitW, statusW, gap := sessionColumns(textWidth)
	limitLabel := bitrate.FormatBitrate(maxBps)
	header := sessionRow([]string{"Path", "Protocol", "Bitrate", "Limit", "Status"}, pathW, protocolW, bitrateW, limitW, statusW, gap)
	header = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Render(header)
	rule := strings.Repeat("─", textWidth)

	lines := []string{header, rule}
	for _, row := range rows {
		bitrateStr := "—"
		if row.BitrateBps != nil {
			bitrateStr = bitrate.FormatBitrate(*row.BitrateBps)
		}
		status := string(row.Status)
		if row.Status == poll.StatusOver && !enforced {
			status = "over (not enforced)"
		}
		line := sessionRow([]string{row.Name, row.SourceType, bitrateStr, limitLabel, status}, pathW, protocolW, bitrateW, limitW, statusW, gap)
		if row.Name == selectedName {
			line = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("230")).Render(line)
		} else {
			statusCell := padClip(status, statusW)
			statusCell = lipgloss.NewStyle().Foreground(statusColor(row, enforced)).Render(statusCell)
			cells := []string{
				padClip(row.Name, pathW),
				padClip(row.SourceType, protocolW),
				padClip(bitrateStr, bitrateW),
				padClip(limitLabel, limitW),
				statusCell,
			}
			line = strings.Join(cells, strings.Repeat(" ", gap))
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func sessionColumns(textWidth int) (pathW int, protocolW int, bitrateW int, limitW int, statusW int, gap int) {
	gap = 2
	protocolW, bitrateW, limitW, statusW = 12, 10, 10, 14
	for {
		pathW = textWidth - (protocolW + bitrateW + limitW + statusW + gap*4)
		if pathW >= 12 {
			return pathW, protocolW, bitrateW, limitW, statusW, gap
		}
		if gap > 1 {
			gap = 1
			continue
		}
		if statusW > 8 {
			statusW = 8
			continue
		}
		if protocolW > 8 {
			protocolW = 8
			continue
		}
		if bitrateW > 8 {
			bitrateW = 8
			limitW = 8
			continue
		}
		if pathW < 4 {
			pathW = 4
		}
		return pathW, protocolW, bitrateW, limitW, statusW, gap
	}
}

func sessionRow(cells []string, pathW int, protocolW int, bitrateW int, limitW int, statusW int, gap int) string {
	widths := []int{pathW, protocolW, bitrateW, limitW, statusW}
	parts := make([]string, len(cells))
	for i, cell := range cells {
		parts[i] = padClip(cell, widths[i])
	}
	return strings.Join(parts, strings.Repeat(" ", gap))
}

func padClip(value string, width int) string {
	if width <= 0 {
		return ""
	}
	clipped := clipTo(value, width)
	extra := width - lipgloss.Width(clipped)
	if extra > 0 {
		clipped += strings.Repeat(" ", extra)
	}
	return clipped
}

func clipTo(value string, width int) string {
	if width <= 0 || value == "" {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	var built strings.Builder
	used := 0
	limit := width - 1
	for _, r := range value {
		runeWidth := lipgloss.Width(string(r))
		if used+runeWidth > limit {
			break
		}
		built.WriteRune(r)
		used += runeWidth
	}
	built.WriteString("…")
	return built.String()
}

func statusColor(row poll.SessionRow, enforced bool) lipgloss.Color {
	switch row.Status {
	case poll.StatusWarming:
		return lipgloss.Color("39")
	case poll.StatusOK:
		return lipgloss.Color("42")
	case poll.StatusOver:
		if enforced {
			return lipgloss.Color("167")
		}
		return lipgloss.Color("214")
	case poll.StatusKicked, poll.StatusKickFailed:
		return lipgloss.Color("167")
	default:
		return lipgloss.Color("243")
	}
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

func viewersPanel(state DashboardState, _ Handlers, textWidth int) string {
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
	lines = append(lines, lipgloss.NewStyle().Width(textWidth).Render(
		lipgloss.JoinVertical(lipgloss.Top,
			fmt.Sprintf("Filter paths... (query: %q)", state.ViewerUi.FilterQuery),
			clipTo(fmt.Sprintf("%s  |  %d/%d paths  |  %s",
				sessionCountLabel, len(preparedGroups), totalPathCount, serverOutboundLabel), textWidth),
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
			lines = append(lines, panelStyle.Width(textWidth-2).Render(strings.Join(grpLines, "\n")))
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
