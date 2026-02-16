package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"boxy/internal/config"
	"boxy/internal/manager"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewMode int

const (
	viewNormal viewMode = iota
	viewSearch
	viewInfo
	viewConfirm
)

type confirmAction int

const (
	confirmInstall confirmAction = iota
	confirmUninstall
)

type packageItem struct {
	info       manager.PackageInfo
	bookmarked bool
}

type Model struct {
	mgr        manager.PackageManager
	cfg        *config.Config
	keys       keyMap
	width      int
	height     int
	cursor     int
	scroll     int // scroll offset for viewport
	viewMode   viewMode
	searchInput textinput.Model
	items      []packageItem
	filtered   []packageItem
	infoText   string
	confirmPkg string
	confirmAct confirmAction
	statusMsg  string
	statusErr  bool
	loading    bool
	searching  bool
	manualSet  map[string]bool
	showAll    bool
}

func NewModel(mgr manager.PackageManager, cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "Search packages..."
	ti.CharLimit = 100
	ti.Width = 40

	return Model{
		mgr:        mgr,
		cfg:        cfg,
		keys:       defaultKeyMap(),
		searchInput: ti,
		loading:    true,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadPackages()
}

func (m Model) loadPackages() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// First, get all installed packages (single brew call)
		installed, err := m.mgr.ListInstalled(ctx)
		if err != nil {
			return packagesLoadedMsg{err: err}
		}

		// Build a set of installed package names for quick lookup
		installedSet := make(map[string]bool)
		for _, pkg := range installed {
			installedSet[pkg.Name] = true
		}

		// For bookmarked packages, just create basic info and check against installed set
		var bookmarked []manager.PackageInfo
		for _, pkg := range m.cfg.Packages {
			bookmarked = append(bookmarked, manager.PackageInfo{
				Name:      pkg,
				Installed: installedSet[pkg],
			})
		}

		manual, _ := m.mgr.ListManuallyInstalled(ctx)

		return packagesLoadedMsg{bookmarked: bookmarked, installed: installed, manual: manual}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorVisible()
		return m, nil

	case packagesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
			m.statusErr = true
		}
		m.manualSet = make(map[string]bool)
		for _, pkg := range msg.manual {
			m.manualSet[pkg.Name] = true
		}
		m.buildItemList(msg.bookmarked, msg.installed)
		return m, nil

	case searchResultsMsg:
		m.searching = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Search error: %v", msg.err)
			m.statusErr = true
			return m, nil
		}
		m.filtered = make([]packageItem, 0, len(msg.results))
		for _, info := range msg.results {
			m.filtered = append(m.filtered, packageItem{
				info:       info,
				bookmarked: m.cfg.IsBookmarked(info.Name),
			})
		}
		m.cursor = 0
		m.scroll = 0
		return m, nil

	case packageInfoMsg:
		if msg.err != nil {
			m.infoText = fmt.Sprintf("Error loading info: %v", msg.err)
		} else {
			m.infoText = formatInfo(msg.info)
		}
		m.viewMode = viewInfo
		return m, nil

	case installResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Install failed: %v", msg.err)
			m.statusErr = true
		} else {
			m.statusMsg = fmt.Sprintf("Installed %s", msg.pkg)
			m.statusErr = false
			m.updateInstallStatus(msg.pkg, true)
		}
		m.viewMode = viewNormal
		return m, nil

	case uninstallResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Uninstall failed: %v", msg.err)
			m.statusErr = true
		} else {
			m.statusMsg = fmt.Sprintf("Uninstalled %s", msg.pkg)
			m.statusErr = false
			m.updateInstallStatus(msg.pkg, false)
		}
		m.viewMode = viewNormal
		return m, nil

	case bookmarkToggledMsg:
		if msg.bookmarked {
			m.statusMsg = fmt.Sprintf("Bookmarked %s", msg.pkg)
		} else {
			m.statusMsg = fmt.Sprintf("Removed bookmark for %s", msg.pkg)
		}
		m.statusErr = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.viewMode {
	case viewSearch:
		return m.handleSearchKey(msg)
	case viewInfo:
		return m.handleInfoKey(msg)
	case viewConfirm:
		return m.handleConfirmKey(msg)
	default:
		return m.handleNormalKey(msg)
	}
}

func (m *Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "q":
		return m, tea.Quit

	case msg.String() == "up" || msg.String() == "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}

	case msg.String() == "down" || msg.String() == "j":
		items := m.visibleItems()
		if m.cursor < len(items)-1 {
			m.cursor++
			m.ensureCursorVisible()
		}

	case msg.String() == "/":
		m.viewMode = viewSearch
		m.searchInput.Focus()
		return m, textinput.Blink

	case msg.String() == "esc":
		if m.filtered != nil {
			// Add any newly bookmarked packages to the main items list
			m.mergeBookmarkedItems()
			m.filtered = nil
			m.searchInput.SetValue("")
			m.cursor = 0
			m.scroll = 0
		}

	case msg.String() == "enter":
		items := m.visibleItems()
		if len(items) > 0 && m.cursor < len(items) {
			pkg := items[m.cursor].info.Name
			m.viewMode = viewInfo
			m.infoText = "Loading..."
			return m, m.fetchInfo(pkg)
		}

	case msg.String() == "i":
		items := m.visibleItems()
		if len(items) > 0 && m.cursor < len(items) {
			item := items[m.cursor]
			if !item.info.Installed {
				m.confirmPkg = item.info.Name
				m.confirmAct = confirmInstall
				m.viewMode = viewConfirm
			}
		}

	case msg.String() == "u":
		items := m.visibleItems()
		if len(items) > 0 && m.cursor < len(items) {
			item := items[m.cursor]
			if item.info.Installed {
				m.confirmPkg = item.info.Name
				m.confirmAct = confirmUninstall
				m.viewMode = viewConfirm
			}
		}

	case msg.String() == "a":
		m.showAll = !m.showAll
		m.cursor = 0
		m.scroll = 0

	case msg.String() == "b":
		items := m.visibleItems()
		if len(items) > 0 && m.cursor < len(items) {
			pkg := items[m.cursor].info.Name
			bookmarked := m.cfg.ToggleBookmark(pkg)
			m.cfg.Save()
			m.updateBookmarkStatus(pkg, bookmarked)
			return m, func() tea.Msg {
				return bookmarkToggledMsg{pkg: pkg, bookmarked: bookmarked}
			}
		}
	}

	return m, nil
}

func (m *Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.viewMode = viewNormal
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.filtered = nil
		m.cursor = 0
		m.scroll = 0
		return m, nil

	case "enter":
		query := m.searchInput.Value()
		if query != "" {
			m.searching = true
			m.viewMode = viewNormal
			m.searchInput.Blur()
			return m, m.searchPackages(query)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m *Model) handleInfoKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" {
		m.viewMode = viewNormal
		m.infoText = ""
	}
	return m, nil
}

func (m *Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		pkg := m.confirmPkg
		if m.confirmAct == confirmInstall {
			return m, m.installPackage(pkg)
		}
		return m, m.uninstallPackage(pkg)

	case "n", "esc":
		m.viewMode = viewNormal
		m.confirmPkg = ""
	}
	return m, nil
}

func (m Model) searchPackages(query string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		results, err := m.mgr.Search(ctx, query)
		if err != nil {
			return searchResultsMsg{err: err}
		}

		// Get installed list once instead of checking each result individually
		installedList, _ := m.mgr.ListInstalled(ctx)
		installedSet := make(map[string]bool)
		for _, pkg := range installedList {
			installedSet[pkg.Name] = true
		}

		for i := range results {
			results[i].Installed = installedSet[results[i].Name]
		}

		return searchResultsMsg{results: results}
	}
}

func (m Model) fetchInfo(pkg string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		info, err := m.mgr.GetInfo(ctx, pkg)
		return packageInfoMsg{info: info, err: err}
	}
}

func (m Model) installPackage(pkg string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.mgr.Install(ctx, pkg)
		return installResultMsg{pkg: pkg, err: err}
	}
}

func (m Model) uninstallPackage(pkg string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.mgr.Uninstall(ctx, pkg)
		return uninstallResultMsg{pkg: pkg, err: err}
	}
}

func (m *Model) buildItemList(bookmarked, installed []manager.PackageInfo) {
	bookmarkedNames := make(map[string]bool)
	for _, pkg := range m.cfg.Packages {
		bookmarkedNames[pkg] = true
	}

	m.items = nil

	for _, info := range bookmarked {
		m.items = append(m.items, packageItem{
			info:       info,
			bookmarked: true,
		})
	}

	for _, info := range installed {
		if !bookmarkedNames[info.Name] {
			m.items = append(m.items, packageItem{
				info:       info,
				bookmarked: false,
			})
		}
	}

	// Sort all items alphabetically by name
	sort.Slice(m.items, func(i, j int) bool {
		return m.items[i].info.Name < m.items[j].info.Name
	})
}

func (m *Model) updateInstallStatus(pkg string, installed bool) {
	for i := range m.items {
		if m.items[i].info.Name == pkg {
			m.items[i].info.Installed = installed
			break
		}
	}
	for i := range m.filtered {
		if m.filtered[i].info.Name == pkg {
			m.filtered[i].info.Installed = installed
			break
		}
	}
}

func (m *Model) updateBookmarkStatus(pkg string, bookmarked bool) {
	for i := range m.items {
		if m.items[i].info.Name == pkg {
			m.items[i].bookmarked = bookmarked
			break
		}
	}
	for i := range m.filtered {
		if m.filtered[i].info.Name == pkg {
			m.filtered[i].bookmarked = bookmarked
			break
		}
	}
}

// mergeBookmarkedItems adds any bookmarked packages from filtered results
// that aren't already in the main items list, then re-sorts
func (m *Model) mergeBookmarkedItems() {
	if m.filtered == nil {
		return
	}

	// Build set of existing item names
	existing := make(map[string]bool)
	for _, item := range m.items {
		existing[item.info.Name] = true
	}

	// Add bookmarked items from filtered that don't exist in main list
	for _, item := range m.filtered {
		if item.bookmarked && !existing[item.info.Name] {
			m.items = append(m.items, item)
			existing[item.info.Name] = true
		}
	}

	// Re-sort alphabetically
	sort.Slice(m.items, func(i, j int) bool {
		return m.items[i].info.Name < m.items[j].info.Name
	})
}

func (m Model) visibleItems() []packageItem {
	if m.filtered != nil {
		return m.filtered
	}
	if !m.showAll {
		var visible []packageItem
		for _, item := range m.items {
			if item.bookmarked || m.manualSet[item.info.Name] {
				visible = append(visible, item)
			}
		}
		return visible
	}
	return m.items
}

// maxVisibleItems returns how many package items can fit on screen
// accounting for header, search bar, status, and help lines
func (m Model) maxVisibleItems() int {
	// Header (2) + search bar (2) + section header (1) + help (3) + status (1) = 9, use 10 for safety
	overhead := 10
	available := m.height - overhead
	if available < 1 {
		return 1
	}
	return available
}

// ensureCursorVisible adjusts scroll to keep cursor in view
func (m *Model) ensureCursorVisible() {
	maxVisible := m.maxVisibleItems()

	// Cursor above viewport - scroll up
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}

	// Cursor below viewport - scroll down
	if m.cursor >= m.scroll+maxVisible {
		m.scroll = m.cursor - maxVisible + 1
	}
}

func (m Model) View() string {
	if m.loading {
		return "Loading..."
	}

	var b strings.Builder

	// Header
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		titleStyle.Render("boxy"),
		managerStyle.Render(fmt.Sprintf("[%s]", m.mgr.Name())),
	)
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", min(m.width, 60)))
	b.WriteString("\n")

	// Search bar
	if m.viewMode == viewSearch {
		b.WriteString(searchStyle.Render("Search: "))
		b.WriteString(m.searchInput.View())
		b.WriteString("\n")
	} else if m.filtered != nil {
		b.WriteString(searchStyle.Render("Search: "))
		b.WriteString(m.searchInput.Value())
		b.WriteString(dimStyle.Render("  (/ to edit, Esc to clear)"))
		b.WriteString("\n")
	} else {
		b.WriteString(dimStyle.Render("Press / to search"))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Package list
	items := m.visibleItems()
	maxVisible := m.maxVisibleItems()

	if m.searching {
		b.WriteString(headerStyle.Render("SEARCH RESULTS"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  Searching..."))
		b.WriteString("\n")
	} else if m.filtered != nil {
		b.WriteString(headerStyle.Render("SEARCH RESULTS"))
		if len(items) > maxVisible {
			b.WriteString(dimStyle.Render(fmt.Sprintf(" (%d-%d of %d)", m.scroll+1, min(m.scroll+maxVisible, len(items)), len(items))))
		}
		b.WriteString("\n")
		m.renderItemsViewport(&b, items, 0, m.scroll, maxVisible)
	} else {
		if m.showAll {
			b.WriteString(headerStyle.Render("PACKAGES (all)"))
		} else {
			b.WriteString(headerStyle.Render("PACKAGES (manual)"))
		}
		if len(items) > maxVisible {
			b.WriteString(dimStyle.Render(fmt.Sprintf(" (%d-%d of %d)", m.scroll+1, min(m.scroll+maxVisible, len(items)), len(items))))
		}
		b.WriteString("\n")
		m.renderItemsViewport(&b, items, 0, m.scroll, maxVisible)
	}

	// Status message
	if m.statusMsg != "" {
		b.WriteString("\n")
		if m.statusErr {
			b.WriteString(errorStyle.Render(m.statusMsg))
		} else {
			b.WriteString(successStyle.Render(m.statusMsg))
		}
	}

	// Help
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/k up  ↓/j down  i install  u uninstall  b bookmark"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Enter info  / search  a manual/all  q quit"))

	// Modal overlay
	if m.viewMode == viewInfo {
		return m.renderWithModal(b.String(), "Package Info", m.infoText)
	}
	if m.viewMode == viewConfirm {
		action := "install"
		if m.confirmAct == confirmUninstall {
			action = "uninstall"
		}
		msg := fmt.Sprintf("Are you sure you want to %s %s?\n\n[y] Yes  [n] No", action, m.confirmPkg)
		return m.renderWithModal(b.String(), "Confirm", msg)
	}

	return b.String()
}

// renderItemsViewport renders items within the viewport
// offset: the global index of the first item in this slice
// scroll: the current scroll position
// maxItems: maximum items to render
// returns: number of items rendered
func (m Model) renderItemsViewport(b *strings.Builder, items []packageItem, offset, scroll, maxItems int) int {
	rendered := 0
	for i, item := range items {
		globalIdx := offset + i

		// Skip items before the scroll position
		if globalIdx < scroll {
			continue
		}

		// Stop if we've rendered enough items
		if rendered >= maxItems {
			break
		}

		prefix := "  "
		if globalIdx == m.cursor {
			prefix = "> "
		}

		bullet := "●"
		if item.bookmarked {
			bullet = bookmarkStyle.Render("●")
		}

		name := item.info.Name
		if globalIdx == m.cursor {
			name = selectedStyle.Render(name)
		} else {
			name = normalStyle.Render(name)
		}

		desc := item.info.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}
		if desc == "" {
			desc = "-"
		}
		desc = dimStyle.Render(desc)

		status := notInstalledStyle.Render("[ ]")
		if item.info.Installed {
			status = installedStyle.Render("[✓]")
		}

		line := fmt.Sprintf("%s%s %s  %s  %s", prefix, bullet, name, desc, status)
		b.WriteString(line)
		b.WriteString("\n")
		rendered++
	}
	return rendered
}

func (m Model) renderWithModal(bg, title, content string) string {
	lines := strings.Split(bg, "\n")

	modalContent := fmt.Sprintf("%s\n\n%s\n\n%s",
		titleStyle.Render(title),
		content,
		dimStyle.Render("Press Esc to close"),
	)
	modal := modalStyle.Render(modalContent)
	modalLines := strings.Split(modal, "\n")

	startY := (len(lines) - len(modalLines)) / 2
	if startY < 0 {
		startY = 0
	}

	for i, mLine := range modalLines {
		lineIdx := startY + i
		if lineIdx < len(lines) {
			lines[lineIdx] = mLine
		}
	}

	return strings.Join(lines, "\n")
}

func formatInfo(info manager.PackageInfo) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Name: %s\n", info.Name))
	if info.Version != "" {
		b.WriteString(fmt.Sprintf("Version: %s\n", info.Version))
	}
	if info.Description != "" {
		b.WriteString(fmt.Sprintf("Description: %s\n", info.Description))
	}
	status := "Not installed"
	if info.Installed {
		status = "Installed"
	}
	b.WriteString(fmt.Sprintf("Status: %s", status))
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
