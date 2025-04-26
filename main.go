package main

import (
	"context"
	"flag"
	"log"
	"rtbackup/backuper"
	"rtbackup/config"
	"rtbackup/notifier"
	"rtbackup/restic"

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
	noti, err := notifier.NewTGNotifier(c.Notifier.Host, c.Notifier.User, c.Notifier.Password)
	if err != nil {
		logkit.Fatal("failed to create notifier", zap.Error(err))
	}
	opts := make([]backuper.Option, 0, len(c.BackupList)+3)
	for _, item := range c.BackupList {
		opts = append(opts, backuper.WithAddBackupItem(item.Name, item.Path, item.Expr, item.PreHooks, item.PostHooks))
	}
	opts = append(opts,
		backuper.WithNotifier(noti),
		backuper.WithRestic(rst),
		backuper.WithEnableDockerCompose(c.EnableDockerCompose),
	)
	b, err := backuper.New(opts...)
	if err != nil {
		logkit.Fatal("failed to create backuper", zap.Error(err))
	}
	if err := b.Run(context.Background()); err != nil {
		logkit.Fatal("failed to run backuper", zap.Error(err))
	}
}
