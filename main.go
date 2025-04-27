package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"rtbackup/backuper"
	"rtbackup/config"
	"rtbackup/notifier"
	"rtbackup/restic"
	"rtbackup/utils"

	"github.com/xxxsen/common/logger"
	"go.uber.org/zap"
)

var (
	conf = flag.String("config", "config.json", "Path to the configuration file for the backuper service.")
)

func main() {
	flag.Parse()
	c, err := config.Parse(*conf)
	if err != nil {
		log.Fatalf("failed to parse config, path:%s, err:%v", *conf, err)
	}
	logkit := logger.Init(c.LogConfig.File, c.LogConfig.Level, int(c.LogConfig.FileCount), int(c.LogConfig.FileSize), int(c.LogConfig.KeepDays), c.LogConfig.Console)

	rst, err := restic.New(restic.WithAuth(c.Restic.Password), restic.WithRepo(c.Restic.Repo))
	if err != nil {
		logkit.Fatal("failed to create restic client", zap.Error(err))
	}
	var noti notifier.INotifier = notifier.Nop
	if c.Notifier.Enable {
		noti, err = notifier.NewTGNotifier(c.Notifier.Host, c.Notifier.User, c.Notifier.Password)
		if err != nil {
			logkit.Fatal("failed to create notifier", zap.Error(err))
		}
	}
	opts := make([]backuper.Option, 0, len(c.BackupList)+3)
	for _, item := range c.BackupList {
		if len(item.BackupPath) == 0 {
			logkit.Fatal("backup item path is empty", zap.String("name", item.Name))
		}
		if err := fixToAbsPath(item); err != nil {
			logkit.Fatal("failed to fix backup item path", zap.String("name", item.Name), zap.String("path", item.BackupPath), zap.Error(err))
		}
		wrapDockerComposeCheck(c, item)
		opts = append(opts, backuper.WithAddBackupItem(item.Name, item.BackupPath, item.Expr, item.PreHooks, item.PostHooks))
	}
	opts = append(opts,
		backuper.WithNotifier(noti),
		backuper.WithRestic(rst),
		backuper.WithKeepRule(c.Restic.Keep.Last, c.Restic.Keep.Daily, c.Restic.Keep.Weekly, c.Restic.Keep.Monthly, c.Restic.Keep.Yearly),
	)
	b, err := backuper.New(opts...)
	if err != nil {
		logkit.Fatal("failed to create backuper", zap.Error(err))
	}
	if err := b.Run(context.Background()); err != nil {
		logkit.Fatal("failed to run backuper", zap.Error(err))
	}
}

func fixToAbsPath(item *config.BackupItem) error {
	var perr, serr error
	if len(item.BackupPath) > 0 {
		item.BackupPath, perr = filepath.Abs(item.BackupPath)
	}
	if len(item.ServicePath) > 0 {
		item.ServicePath, serr = filepath.Abs(item.ServicePath)
	}
	if perr != nil || serr != nil {
		return fmt.Errorf("fix to abs path failed, path:%s, service_path:%s", item.BackupPath, item.ServicePath)
	}
	return nil
}

func wrapDockerComposeCheck(c *config.Config, item *config.BackupItem) {
	if !c.SwitchConfig.CheckDockerCompose {
		return
	}
	path := item.ServicePath
	if len(path) == 0 {
		path = item.BackupPath
	}
	if !utils.IsFileExists(path, []string{"docker-compose.yml", "docker-compose.yaml"}) {
		return
	}
	item.PreHooks = append([]string{fmt.Sprintf("cd %s && docker compose stop", path)}, item.PreHooks...)
	item.PostHooks = append(item.PostHooks, fmt.Sprintf("cd %s && docker compose restart", path))
}
