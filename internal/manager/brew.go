package manager

import (
	"bufio"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
)

type BrewManager struct{}

func (b *BrewManager) Name() string {
	return "brew"
}

func (b *BrewManager) IsAvailable() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

func (b *BrewManager) Search(ctx context.Context, query string) ([]PackageInfo, error) {
	cmd := exec.CommandContext(ctx, "brew", "search", query)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var results []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "==>") {
			continue
		}
		results = append(results, PackageInfo{
			Name: line,
		})
	}

	return results, scanner.Err()
}

func (b *BrewManager) Command(ctx context.Context, action string, pkg string) *exec.Cmd {
	switch action {
	case "install":
		return exec.CommandContext(ctx, "brew", "install", pkg)
	case "uninstall":
		return exec.CommandContext(ctx, "brew", "uninstall", pkg)
	}
	return exec.CommandContext(ctx, "brew", action, pkg)
}

func (b *BrewManager) NeedsSudo() bool {
	return false
}

func (b *BrewManager) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"install"}, packages...)
	cmd := exec.CommandContext(ctx, "brew", args...)
	return cmd.Run()
}

func (b *BrewManager) Uninstall(ctx context.Context, packages ...string) error {
	args := append([]string{"uninstall"}, packages...)
	cmd := exec.CommandContext(ctx, "brew", args...)
	return cmd.Run()
}

func (b *BrewManager) IsInstalled(ctx context.Context, pkg string) (bool, error) {
	cmd := exec.CommandContext(ctx, "brew", "list", "--formula", pkg)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type brewInfoJSON struct {
	Name    string   `json:"name"`
	Version string   `json:"versions"`
	Desc    string   `json:"desc"`
	Tap     string   `json:"tap"`
	Full    string   `json:"full_name"`
}

type brewInfoVersions struct {
	Stable string `json:"stable"`
}

type brewInfoEntry struct {
	Name     string           `json:"name"`
	FullName string           `json:"full_name"`
	Desc     string           `json:"desc"`
	Versions brewInfoVersions `json:"versions"`
}

func (b *BrewManager) GetInfo(ctx context.Context, pkg string) (PackageInfo, error) {
	cmd := exec.CommandContext(ctx, "brew", "info", "--json=v2", pkg)
	output, err := cmd.Output()
	if err != nil {
		return PackageInfo{}, err
	}

	var result struct {
		Formulae []brewInfoEntry `json:"formulae"`
		Casks    []brewInfoEntry `json:"casks"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return PackageInfo{}, err
	}

	if len(result.Formulae) > 0 {
		f := result.Formulae[0]
		installed, _ := b.IsInstalled(ctx, pkg)
		return PackageInfo{
			Name:        f.Name,
			Version:     f.Versions.Stable,
			Description: f.Desc,
			Installed:   installed,
		}, nil
	}

	if len(result.Casks) > 0 {
		c := result.Casks[0]
		return PackageInfo{
			Name:        c.Name,
			Version:     c.Versions.Stable,
			Description: c.Desc,
			Installed:   false,
		}, nil
	}

	return PackageInfo{Name: pkg}, nil
}

func (b *BrewManager) ListInstalled(ctx context.Context) ([]PackageInfo, error) {
	cmd := exec.CommandContext(ctx, "brew", "list", "--formula", "-1")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var results []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name == "" {
			continue
		}
		results = append(results, PackageInfo{
			Name:      name,
			Installed: true,
		})
	}

	return results, scanner.Err()
}

func (b *BrewManager) ListManuallyInstalled(ctx context.Context) ([]PackageInfo, error) {
	cmd := exec.CommandContext(ctx, "brew", "leaves")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var results []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name == "" {
			continue
		}
		results = append(results, PackageInfo{
			Name:      name,
			Installed: true,
		})
	}

	return results, scanner.Err()
}
