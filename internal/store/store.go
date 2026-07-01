package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	appLog "github.com/gustmrg/lofi/internal/log"
)

type Saved struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SourceRef   string `json:"source_ref"`
}

type Config struct {
	Volume int `json:"volume"`
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}
	return filepath.Join(home, ".lofi", "stations.json"), nil
}

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}
	return filepath.Join(home, ".lofi", "config.json"), nil
}

func Load() ([]Saved, error) {
	logger := appLog.For("store")
	p, err := Path()
	if err != nil {
		logger.Warn("locate stations path failed", "op", "load_stations", "err", err)
		return nil, err
	}
	logger.Debug("loading stations", "op", "load_stations", "path", p)
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			logger.Debug("stations file missing", "op", "load_stations", "path", p)
			return nil, nil
		}
		logger.Warn("read stations failed", "op", "load_stations", "path", p, "err", err)
		return nil, fmt.Errorf("read %s: %w", p, err)
	}
	var out []Saved
	if err := json.Unmarshal(data, &out); err != nil {
		logger.Warn("parse stations failed", "op", "load_stations", "path", p, "err", err)
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	logger.Debug("loaded stations", "op", "load_stations", "path", p, "count", len(out))
	return out, nil
}

func Save(stations []Saved) error {
	logger := appLog.For("store")
	p, err := Path()
	if err != nil {
		logger.Warn("locate stations path failed", "op", "save_stations", "err", err)
		return err
	}
	logger.Debug("saving stations", "op", "save_stations", "path", p, "count", len(stations))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		logger.Warn("create config directory failed", "op", "save_stations", "path", filepath.Dir(p), "err", err)
		return fmt.Errorf("create dir: %w", err)
	}
	data, err := json.MarshalIndent(stations, "", "  ")
	if err != nil {
		logger.Warn("encode stations failed", "op", "save_stations", "err", err)
		return fmt.Errorf("encode: %w", err)
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		logger.Warn("write stations failed", "op", "save_stations", "path", tmp, "err", err)
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		logger.Warn("rename stations failed", "op", "save_stations", "from", tmp, "to", p, "err", err)
		return fmt.Errorf("rename %s: %w", p, err)
	}
	logger.Debug("saved stations", "op", "save_stations", "path", p, "count", len(stations))
	return nil
}

func LoadConfig() (*Config, error) {
	logger := appLog.For("store")
	p, err := ConfigPath()
	if err != nil {
		logger.Warn("locate config path failed", "op", "load_config", "err", err)
		return nil, err
	}
	logger.Debug("loading config", "op", "load_config", "path", p)
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			logger.Debug("config file missing", "op", "load_config", "path", p)
			return nil, nil
		}
		logger.Warn("read config failed", "op", "load_config", "path", p, "err", err)
		return nil, fmt.Errorf("read %s: %w", p, err)
	}
	var out Config
	if err := json.Unmarshal(data, &out); err != nil {
		logger.Warn("parse config failed", "op", "load_config", "path", p, "err", err)
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	logger.Debug("loaded config", "op", "load_config", "path", p)
	return &out, nil
}

func SaveConfig(cfg Config) error {
	logger := appLog.For("store")
	p, err := ConfigPath()
	if err != nil {
		logger.Warn("locate config path failed", "op", "save_config", "err", err)
		return err
	}
	logger.Debug("saving config", "op", "save_config", "path", p)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		logger.Warn("create config directory failed", "op", "save_config", "path", filepath.Dir(p), "err", err)
		return fmt.Errorf("create dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		logger.Warn("encode config failed", "op", "save_config", "err", err)
		return fmt.Errorf("encode: %w", err)
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		logger.Warn("write config failed", "op", "save_config", "path", tmp, "err", err)
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		logger.Warn("rename config failed", "op", "save_config", "from", tmp, "to", p, "err", err)
		return fmt.Errorf("rename %s: %w", p, err)
	}
	logger.Debug("saved config", "op", "save_config", "path", p)
	return nil
}
