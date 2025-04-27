package config

import (
	"encoding/json"
	"os"

	"github.com/xxxsen/common/logger"
)

type BackupItem struct {
	Name        string   `json:"name"`
	Expr        string   `json:"expr"`         // cron expression
	BackupPath  string   `json:"backup_path"`  // path to backup
	ServicePath string   `json:"service_path"` // docker compose 探测的目录, 如果没指定, 则使用path进行探测
	PreHooks    []string `json:"pre_hooks"`    // commands to run before backup
	PostHooks   []string `json:"post_hooks"`   // commands to run after backup
}

type Notifier struct {
	Enable   bool   `json:"enable"`   // whether to enable notification
	Host     string `json:"host"`     // notification service host
	User     string `json:"user"`     // user for notification service
	Password string `json:"password"` // password for notification service
}

type ResticKeep struct {
	Last    int `json:"last"`    // keep last N backups
	Daily   int `json:"daily"`   // keep daily backups for N days
	Weekly  int `json:"weekly"`  // keep weekly backups for N weeks
	Monthly int `json:"monthly"` // keep monthly backups for N months
	Yearly  int `json:"yearly"`  // keep yearly backups for N years
}

type Restic struct {
	Repo     string     `json:"repo"`     // restic repository path
	Password string     `json:"password"` // restic repository password
	Keep     ResticKeep `json:"keep"`     // restic keep policy
}

type Config struct {
	BackupList   []*BackupItem    `json:"backup_list"`
	Notifier     Notifier         `json:"notifier"`
	Restic       Restic           `json:"restic"`        // restic configuration
	LogConfig    logger.LogConfig `json:"log_config"`    // logging configuration
	SwitchConfig SwitchConfig     `json:"switch_config"` // switch configuration for enabling/disabling features
}

type SwitchConfig struct {
	CheckDockerCompose bool `json:"check_docker_compose"` // whether to enable docker compose for backup
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
