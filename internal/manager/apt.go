package manager

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type AptManager struct{}

func (a *AptManager) Name() string {
	return "apt"
}

func (a *AptManager) IsAvailable() bool {
	_, err := exec.LookPath("apt")
	return err == nil
}

func (a *AptManager) Search(ctx context.Context, query string) ([]PackageInfo, error) {
	cmd := exec.CommandContext(ctx, "apt-cache", "search", query)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var results []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " - ", 2)
		info := PackageInfo{
			Name: parts[0],
		}
		if len(parts) > 1 {
			info.Description = parts[1]
		}
		results = append(results, info)
	}

	return results, scanner.Err()
}

func (a *AptManager) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"apt-get", "install", "-y"}, packages...)
	cmd := exec.CommandContext(ctx, "sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("sudo apt-get install failed (try running 'sudo -v' first to cache credentials): %w", err)
	}
	return nil
}

func (a *AptManager) Uninstall(ctx context.Context, packages ...string) error {
	args := append([]string{"apt-get", "remove", "-y"}, packages...)
	cmd := exec.CommandContext(ctx, "sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("sudo apt-get remove failed (try running 'sudo -v' first to cache credentials): %w", err)
	}
	return nil
}

func (a *AptManager) IsInstalled(ctx context.Context, pkg string) (bool, error) {
	cmd := exec.CommandContext(ctx, "dpkg-query", "-W", "-f=${Status}", pkg)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return strings.Contains(string(output), "install ok installed"), nil
}

func (a *AptManager) GetInfo(ctx context.Context, pkg string) (PackageInfo, error) {
	cmd := exec.CommandContext(ctx, "apt-cache", "show", pkg)
	output, err := cmd.Output()
	if err != nil {
		return PackageInfo{}, err
	}

	info := PackageInfo{Name: pkg}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Package: ") {
			info.Name = strings.TrimPrefix(line, "Package: ")
		} else if strings.HasPrefix(line, "Version: ") {
			info.Version = strings.TrimPrefix(line, "Version: ")
		} else if strings.HasPrefix(line, "Description: ") {
			info.Description = strings.TrimPrefix(line, "Description: ")
		}
	}

	installed, _ := a.IsInstalled(ctx, pkg)
	info.Installed = installed

	return info, scanner.Err()
}

func (a *AptManager) ListInstalled(ctx context.Context) ([]PackageInfo, error) {
	cmd := exec.CommandContext(ctx, "dpkg-query", "-W", "-f=${Package}\n")
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

func (a *AptManager) ListManuallyInstalled(ctx context.Context) ([]PackageInfo, error) {
	cmd := exec.CommandContext(ctx, "apt-mark", "showmanual")
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
