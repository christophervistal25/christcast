package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Scope struct {
	Path string `toml:"path"`
}

type Config struct {
	Scopes          []Scope  `toml:"scopes"`
	Excludes        []string `toml:"excludes"`
	FollowSymlinks  bool     `toml:"follow_symlinks"`
	IncludeHidden   bool     `toml:"include_hidden"`
	CrossDevice     bool     `toml:"cross_device"`
	MaxResults      int      `toml:"max_results"`
	Hotkey          string   `toml:"hotkey"`
}

func Default() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Scopes: []Scope{{Path: home}},
		Excludes: []string{
			"node_modules", ".git", ".cache", "target", "vendor",
			"dist", "build", ".venv", "__pycache__", ".gradle", ".m2",
			"Library", ".npm", ".yarn", ".pnpm-store", ".rustup", ".cargo",
		},
		FollowSymlinks: false,
		IncludeHidden:  false,
		CrossDevice:    false,
		MaxResults:     50,
		Hotkey:         "ctrl+space",
	}
}

func ConfigDir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "chriscast")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "chriscast")
}

func DataDir() string {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "chriscast")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "chriscast")
}

func StateDir() string {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "chriscast")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "chriscast")
}

func Path() string { return filepath.Join(ConfigDir(), "config.toml") }

func Load() (*Config, error) {
	p := Path()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		c := Default()
		if err := Save(c); err != nil {
			return nil, err
		}
		return c, nil
	}
	c := Default()
	if _, err := toml.DecodeFile(p, c); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return c, nil
}

func Save(c *Config) error {
	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		return err
	}
	f, err := os.Create(Path())
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}
