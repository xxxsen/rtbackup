package backuper

import (
	"rtbackup/notifier"
	"rtbackup/restic"
)

type backupItem struct {
	Name     string
	Path     string
	Expr     string
	PreRun   []string
	AfterRun []string
}

type config struct {
	backupList          []backupItem
	resitcer            restic.IResitc
	notifier            notifier.INotifier
	enableDockerCompose bool
}

type Option func(c *config)

func WithAddBackupItem(name string, path string, expr string, prerun []string, afterrun []string) Option {
	return func(c *config) {
		c.backupList = append(c.backupList, backupItem{
			Name:     name,
			Path:     path,
			Expr:     expr,
			PreRun:   prerun,
			AfterRun: afterrun,
		})
	}
}

func WithRestic(r restic.IResitc) Option {
	return func(c *config) {
		c.resitcer = r
	}
}
func WithNotifier(n notifier.INotifier) Option {
	return func(c *config) {
		c.notifier = n
	}
}
func WithEnableDockerCompose(enable bool) Option {
	return func(c *config) {
		c.enableDockerCompose = enable
	}
}
