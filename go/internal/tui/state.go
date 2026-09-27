package tui

import (
	"slices"
	"strings"

	"mio9/mtx-monitor/internal/config"
	"mio9/mtx-monitor/internal/poll"
)

// DashboardSection identifies a tab in the dashboard.
type DashboardSection string

const (
	SectionEnforced   DashboardSection = "enforced"
	SectionOther      DashboardSection = "other"
	SectionViewers    DashboardSection = "viewers"
)

var DASHBOARD_SECTIONS = []DashboardSection{SectionEnforced, SectionOther, SectionViewers}

// ViewerUiState holds viewer-panel UI state.
type ViewerUiState struct {
	FilterQuery    string
	CollapsedPaths []string
	PinnedPaths    []string
	SelectedPathName *string
}

// DashboardState is the full TUI application state.
type DashboardState struct {
	Config        config.Config
	Snapshot      poll.PollSnapshot
	LastUpdatedMs *int64               // Unix ms, nil = never
	ActiveSection DashboardSection
	SelectedKeys  map[string]string    // section → row name (enforced/other/viewers)
	ConfirmKick   *poll.SessionRow
	ActionMessage *string
	ViewerUi      ViewerUiState
}

// createInitialState returns the initial dashboard state.
func CreateInitialState(cfg config.Config) DashboardState {
	return DashboardState{
		Config: cfg,
		Snapshot: poll.PollSnapshot{
			Enforced: []poll.SessionRow{},
			Other:    []poll.SessionRow{},
			Viewers:  []poll.ViewerGroup{},
		},
		LastUpdatedMs: nil,
		ActiveSection: SectionEnforced,
		SelectedKeys: map[string]string{
			"enforced": "",
			"other":    "",
			"viewers":  "",
		},
		ConfirmKick:   nil,
		ActionMessage: nil,
		ViewerUi:      ViewerUiState{},
	}
}

// PrepareViewerGroups filters and sorts viewer groups by query, pin order, then name.
func PrepareViewerGroups(groups []poll.ViewerGroup, ui ViewerUiState) []poll.ViewerGroup {
	query := strings.ToLower(strings.TrimSpace(ui.FilterQuery))
	var filtered []poll.ViewerGroup
	for _, g := range groups {
		if query == "" || strings.Contains(strings.ToLower(g.PathName), query) {
			filtered = append(filtered, g)
		}
	}

	pinnedSet := make(map[string]struct{}, len(ui.PinnedPaths))
	for _, p := range ui.PinnedPaths {
		pinnedSet[p] = struct{}{}
	}
	pinnedOrder := make(map[string]int, len(ui.PinnedPaths))
	for i, p := range ui.PinnedPaths {
		pinnedOrder[p] = i
	}

	slices.SortFunc(filtered, func(a, b poll.ViewerGroup) int {
		aPinned := a.PathName == "" || pinnedSet[a.PathName] != struct{}{}
		bPinned := b.PathName == "" || pinnedSet[b.PathName] != struct{}{}
		if aPinned != bPinned {
			if aPinned {
				return -1
			}
			return 1
		}
		if aPinned && bPinned {
			return pinnedOrder[a.PathName] - pinnedOrder[b.PathName]
		}
		return strings.Compare(a.PathName, b.PathName)
	})

	return filtered
}

// IsViewerPathCollapsed checks if a path is collapsed.
func IsViewerPathCollapsed(ui ViewerUiState, pathName string) bool {
	return slices.Contains(ui.CollapsedPaths, pathName)
}

// IsViewerPathPinned checks if a path is pinned.
func IsViewerPathPinned(ui ViewerUiState, pathName string) bool {
	return slices.Contains(ui.PinnedPaths, pathName)
}

// ToggleViewerPathCollapsed adds/removes a path from collapsed list.
func ToggleViewerPathCollapsed(ui ViewerUiState, pathName string) ViewerUiState {
	if slices.Contains(ui.CollapsedPaths, pathName) {
		ui.CollapsedPaths = slices.DeleteFunc(ui.CollapsedPaths, func(s string) bool { return s == pathName })
	} else {
		ui.CollapsedPaths = append(ui.CollapsedPaths, pathName)
	}
	return ui
}

// ToggleViewerPathPinned adds/removes a path from pinned list.
func ToggleViewerPathPinned(ui ViewerUiState, pathName string) ViewerUiState {
	if slices.Contains(ui.PinnedPaths, pathName) {
		ui.PinnedPaths = slices.DeleteFunc(ui.PinnedPaths, func(s string) bool { return s == pathName })
	} else {
		ui.PinnedPaths = append(ui.PinnedPaths, pathName)
	}
	return ui
}

// PruneViewerUi removes stale collapsed/pinned/selected entries for removed paths.
func PruneViewerUi(ui ViewerUiState, groups []poll.ViewerGroup) ViewerUiState {
	pathNames := make(map[string]struct{}, len(groups))
	for _, g := range groups {
		pathNames[g.PathName] = struct{}{}
	}

	ui.CollapsedPaths = slices.DeleteFunc(ui.CollapsedPaths, func(s string) bool { return pathNames[s] == struct{}{} })
	ui.PinnedPaths = slices.DeleteFunc(ui.PinnedPaths, func(s string) bool { return pathNames[s] == struct{}{} })

	visible := PrepareViewerGroups(groups, ui)
	if ui.SelectedPathName != nil && pathNames[*ui.SelectedPathName] == struct{}{} {
		// keep selected
	} else {
		ui.SelectedPathName = nil
		if len(visible) > 0 {
			pn := visible[0].PathName
			ui.SelectedPathName = &pn
		}
	}

	return ui
}

// IsPublisherSection returns true if section is enforced or other.
func IsPublisherSection(section DashboardSection) bool {
	return section != SectionViewers
}

// CycleDashboardSection cycles through sections by step (+1 or -1).
func CycleDashboardSection(current DashboardSection, step int) DashboardSection {
	idx := -1
	for i, s := range DASHBOARD_SECTIONS {
		if s == current {
			idx = i
			break
		}
	}
	if idx < 0 {
		return current
	}
	nextIdx := (idx + step + len(DASHBOARD_SECTIONS)) % len(DASHBOARD_SECTIONS)
	return DASHBOARD_SECTIONS[nextIdx]
}

// GetSectionRows returns publisher rows for the given section.
func GetSectionRows(state DashboardState, section DashboardSection) []poll.SessionRow {
	switch section {
	case SectionEnforced:
		return state.Snapshot.Enforced
	case SectionOther:
		return state.Snapshot.Other
	default:
		return nil
	}
}

// FlattenViewers returns a flat list of all viewer rows across groups.
func FlattenViewers(snapshot poll.PollSnapshot) []poll.ViewerRow {
	var result []poll.ViewerRow
	for _, g := range snapshot.Viewers {
		result = append(result, g.Readers...)
	}
	return result
}

// GetSelectedRow returns the currently selected publisher row, or nil.
func GetSelectedRow(state DashboardState) *poll.SessionRow {
	if !IsPublisherSection(state.ActiveSection) {
		return nil
	}
	selectedKey := state.SelectedKeys[string(state.ActiveSection)]
	if selectedKey == "" {
		return nil
	}
	for _, row := range GetSectionRows(state, state.ActiveSection) {
		if row.Name == selectedKey {
			return &row
		}
	}
	return nil
}

// PruneSelections prunes stale publisher selections and defaults to first available.
func PruneSelections(keys map[string]string, snapshot poll.PollSnapshot) map[string]string {
	prunePublisher := func(section string, rows []poll.SessionRow) string {
		current := keys[section]
		for _, row := range rows {
			if row.Name == current {
				return current
			}
		}
		if len(rows) > 0 {
			return rows[0].Name
		}
		return ""
	}

	return map[string]string{
		"enforced": prunePublisher("enforced", snapshot.Enforced),
		"other":    prunePublisher("other", snapshot.Other),
		"viewers":  keys["viewers"],
	}
}
