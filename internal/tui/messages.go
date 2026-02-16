package tui

import "boxy/internal/manager"

type packagesLoadedMsg struct {
	bookmarked []manager.PackageInfo
	installed  []manager.PackageInfo
	manual     []manager.PackageInfo
	err        error
}

type searchResultsMsg struct {
	results []manager.PackageInfo
	err     error
}

type packageInfoMsg struct {
	info manager.PackageInfo
	err  error
}

type installResultMsg struct {
	pkg string
	err error
}

type uninstallResultMsg struct {
	pkg string
	err error
}

type bookmarkToggledMsg struct {
	pkg        string
	bookmarked bool
}
