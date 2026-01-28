package manager

import "context"

type PackageInfo struct {
	Name        string
	Version     string
	Description string
	Installed   bool
}

type PackageManager interface {
	Name() string
	IsAvailable() bool
	Search(ctx context.Context, query string) ([]PackageInfo, error)
	Install(ctx context.Context, packages ...string) error
	Uninstall(ctx context.Context, packages ...string) error
	IsInstalled(ctx context.Context, pkg string) (bool, error)
	GetInfo(ctx context.Context, pkg string) (PackageInfo, error)
	ListInstalled(ctx context.Context) ([]PackageInfo, error)
}
