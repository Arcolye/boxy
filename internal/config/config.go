package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Packages []string `yaml:"packages"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "boxy", "packages.yaml"), nil
}

func Load() (*Config, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Packages: []string{}}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Packages == nil {
		cfg.Packages = []string{}
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (c *Config) IsBookmarked(pkg string) bool {
	for _, p := range c.Packages {
		if p == pkg {
			return true
		}
	}
	return false
}

func (c *Config) AddBookmark(pkg string) {
	if !c.IsBookmarked(pkg) {
		c.Packages = append(c.Packages, pkg)
	}
}

func (c *Config) RemoveBookmark(pkg string) {
	for i, p := range c.Packages {
		if p == pkg {
			c.Packages = append(c.Packages[:i], c.Packages[i+1:]...)
			return
		}
	}
}

func (c *Config) ToggleBookmark(pkg string) bool {
	if c.IsBookmarked(pkg) {
		c.RemoveBookmark(pkg)
		return false
	}
	c.AddBookmark(pkg)
	return true
}
