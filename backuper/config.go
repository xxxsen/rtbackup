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

type keepRule struct {
	Last    int `json:"last"`    // keep last N backups
	Daily   int `json:"daily"`   // keep daily backups for N days
	Weekly  int `json:"weekly"`  // keep weekly backups for N weeks
	Monthly int `json:"monthly"` // keep monthly backups for N months
	Yearly  int `json:"yearly"`  // keep yearly backups for N years

}

type config struct {
	backupList []backupItem
	keepRule   keepRule
	resitcer   restic.IResitc
	notifier   notifier.INotifier
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

func WithKeepRule(last, daily, weekly, monthly, yearly int) Option {
	return func(c *config) {
		c.keepRule = keepRule{
			Last:    last,
			Daily:   daily,
			Weekly:  weekly,
			Monthly: monthly,
			Yearly:  yearly,
		}
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
