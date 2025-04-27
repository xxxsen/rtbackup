package backuper

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"rtbackup/notifier"
	"runtime/debug"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/xxxsen/common/logutil"
	"github.com/xxxsen/common/trace"
	"go.uber.org/zap"
)

type IBackuper interface {
	Run(ctx context.Context) error
}

type backuperImpl struct {
	c *config
}

func New(opts ...Option) (IBackuper, error) {
	c := &config{}
	for _, opt := range opts {
		opt(c)
	}
	return &backuperImpl{
		c: c,
	}, nil
}

func (b *backuperImpl) Run(ctx context.Context) error {
	c := cron.New(cron.WithSeconds())
	for _, item := range b.c.backupList {
		item := item
		logutil.GetLogger(ctx).Info("add backup item", zap.String("name", item.Name), zap.String("path", item.Path), zap.String("expr", item.Expr))
		if _, err := c.AddFunc(item.Expr, b.runOne(ctx, &item)); err != nil {
			return fmt.Errorf("add cron func failed, item:%+v, err:%w", item, err)
		}
	}
	c.Run()
	return nil
}

func (b *backuperImpl) runOne(ctx context.Context, item *backupItem) func() {
	ctx = trace.WithTraceId(ctx, fmt.Sprintf("BK:%s", item.Name))
	logger := logutil.GetLogger(ctx).With(zap.String("name", item.Name), zap.String("path", item.Path))
	return func() {
		logger.Info("backup item start")
		start := time.Now()
		err := b.runItemBackup(ctx, item)
		end := time.Now()
		b.doNotify(ctx, item, start, end, err)
		logger.Error("backup item finish", zap.Error(err), zap.Duration("cost", end.Sub(start)))
	}
}

func (b *backuperImpl) doNotify(ctx context.Context, item *backupItem, start, end time.Time, err error) {
	if b.c.notifier == nil {
		return
	}
	nt := &notifier.Notification{
		Title:     fmt.Sprintf("Backup %s Report", item.Name),
		Path:      item.Path,
		Start:     start.UnixMilli(),
		End:       end.UnixMilli(),
		IsSuccess: err == nil,
	}
	if err != nil {
		nt.Errmsg = err.Error()
	}
	if err := b.c.notifier.Notify(ctx, nt); err != nil {
		logutil.GetLogger(ctx).Error("notify failed", zap.Error(err), zap.String("name", item.Name), zap.Any("msg", *nt))
	}
}

func (b *backuperImpl) runItemBackup(ctx context.Context, item *backupItem) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("run item backup panic, name:%s, panic:%v, stack:%s", item.Name, r, string(debug.Stack()))
		}
	}()
	defer func() {
		if b.c.keepRule.Last == 0 && b.c.keepRule.Daily == 0 && b.c.keepRule.Weekly == 0 &&
			b.c.keepRule.Monthly == 0 && b.c.keepRule.Yearly == 0 {
			return
		}
		if e := b.c.resitcer.Forget(ctx, b.c.keepRule.Last, b.c.keepRule.Daily, b.c.keepRule.Weekly,
			b.c.keepRule.Monthly, b.c.keepRule.Yearly); e != nil {
			err = fmt.Errorf("do backup forget failed, name:%s, err:%w", item.Name, e)
		}
	}()

	preHooks, afterHooks := item.PreRun, item.AfterRun
	defer func() {
		if e := b.runCmds(ctx, item, afterHooks); e != nil { //after无论如何都要执行
			err = fmt.Errorf("after-run command failed, name:%s, err:%w", item.Name, e)
		}
	}()
	if err := b.runCmds(ctx, item, preHooks); err != nil {
		return fmt.Errorf("pre-run command failed, name:%s, err:%w", item.Name, err)
	}
	if err := b.doBackup(ctx, item); err != nil {
		return fmt.Errorf("backup failed, name:%s, err:%w", item.Name, err)
	}
	return nil
}

func (b *backuperImpl) doBackup(ctx context.Context, item *backupItem) error {
	if err := b.c.resitcer.Backup(ctx, item.Path); err != nil {
		return fmt.Errorf("backup failed, path:%s, err:%w", item.Path, err)
	}
	return nil
}

func (b *backuperImpl) runCmds(ctx context.Context, item *backupItem, cmds []string) error {
	if len(cmds) == 0 {
		return nil
	}
	var retErr error
	for _, cmd := range cmds {
		if err := b.runCmd(ctx, item.Path, cmd); err != nil {
			retErr = err
		}
	}
	return retErr
}

func (b *backuperImpl) runCmd(ctx context.Context, workdir string, cmdstr string) error {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", cmdstr)
	stderr := bytes.Buffer{}
	stdout := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = workdir
	start := time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("exec cmd failed, cmd:%s, err:%w, errmsg:%s", cmdstr, err, stderr.String())
	}
	logutil.GetLogger(ctx).Info("run command success",
		zap.String("cmd", cmdstr),
		zap.String("workdir", workdir),
		zap.String("stdout", stdout.String()),
		zap.String("stderr", stderr.String()),
		zap.Duration("cost", time.Since(start)))
	return nil
}
