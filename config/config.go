package config

import (
	"encoding/json"
	"os"

	"github.com/xxxsen/common/logger"
)

type BackupItem struct {
	Name      string   `json:"name"`
	Expr      string   `json:"expr"`       // cron expression
	Path      string   `json:"path"`       // path to backup
	PreHooks  []string `json:"pre_hooks"`  // commands to run before backup
	PostHooks []string `json:"post_hooks"` // commands to run after backup
}

type Notifier struct {
	Host     string `json:"host"`     // notification service host
	User     string `json:"user"`     // user for notification service
	Password string `json:"password"` // password for notification service
}

type Restic struct {
	Repo     string `json:"repo"`     // restic repository path
	Password string `json:"password"` // restic repository password
}

type Config struct {
	BackupList          []BackupItem     `json:"backup_list"`
	Notifier            Notifier         `json:"notifier"`
	EnableDockerCompose bool             `json:"enable_docker_compose"` // whether to enable docker compose for backup
	Restic              Restic           `json:"restic"`                // restic configuration
	LogConfig           logger.LogConfig `json:"log_config"`            // logging configuration
}

func Parse(f string) (*Config, error) {
	c := &Config{}
	raw, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, c); err != nil {
		return nil, err
	}
	return c, nil
}
