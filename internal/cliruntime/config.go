package cliruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 本地 CLI 配置（用户身份等）。
type Config struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName,omitempty"`
}

// LinkConfig 与已部署 Regulus 实例的关联。
type LinkConfig struct {
	APIURL string `json:"apiUrl"`
	UserID string `json:"userId"`
}

func loadJSON(path string, dest any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}

func saveJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (rt *Runtime) loadConfig() error {
	rt.config = Config{UserID: "default"}
	if _, err := os.Stat(rt.paths.ConfigPath); err != nil {
		return nil
	}
	var cfg Config
	if err := loadJSON(rt.paths.ConfigPath, &cfg); err != nil {
		return fmt.Errorf("读取 config.json 失败: %w", err)
	}
	if cfg.UserID != "" {
		rt.config = cfg
	}
	return nil
}

func (rt *Runtime) loadLink() error {
	rt.link = nil
	if _, err := os.Stat(rt.paths.LinkPath); err != nil {
		return nil
	}
	var link LinkConfig
	if err := loadJSON(rt.paths.LinkPath, &link); err != nil {
		return fmt.Errorf("读取 link.json 失败: %w", err)
	}
	if link.APIURL == "" {
		return nil
	}
	if link.UserID == "" {
		link.UserID = rt.config.UserID
	}
	rt.link = &link
	return nil
}

// SaveLink 写入 link.json。
func (rt *Runtime) SaveLink(link LinkConfig) error {
	if err := os.MkdirAll(rt.paths.RegulusDir, 0o755); err != nil {
		return err
	}
	if link.UserID == "" {
		link.UserID = rt.config.UserID
	}
	rt.link = &link
	return saveJSON(rt.paths.LinkPath, link)
}

// UserID 当前 CLI 用户 ID。
func (rt *Runtime) UserID() string {
	if rt.link != nil && rt.link.UserID != "" {
		return rt.link.UserID
	}
	if rt.config.UserID != "" {
		return rt.config.UserID
	}
	return "default"
}

// Linked 是否已关联远程实例。
func (rt *Runtime) Linked() bool {
	return rt.link != nil && rt.link.APIURL != ""
}
