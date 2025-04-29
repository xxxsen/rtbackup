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

type backupContext struct {
	Item        *backupItem
	NextRunTime time.Time
}

type IBackuper interface {
	Run(ctx context.Context) error
}

type bkImpl struct {
	c        *config
	bctxList []*backupContext
}

func New(opts ...Option) (IBackuper, error) {
	c := &config{}
	for _, opt := range opts {
		opt(c)
	}
	b := &bkImpl{
		c: c,
	}
	bctxList := make([]*backupContext, 0, len(c.backupList))
	for i := range c.backupList {
		backItem := c.backupList[i]
		next, err := b.calcNextRunTime(backItem.Expr)
		if err != nil {
			return nil, fmt.Errorf("invalid expr:%s, name:%s", backItem.Expr, backItem.Name)
		}
		bctx := &backupContext{
			Item:        &backItem,
			NextRunTime: next,
		}
		bctxList = append(bctxList, bctx)
		logutil.GetLogger(context.Background()).Info("add backup item", zap.String("name", backItem.Name), zap.String("expr", backItem.Expr),
			zap.String("path", backItem.Path), zap.Time("next_run_time", next))
	}
	b.bctxList = bctxList
	return b, nil
}

func (b *bkImpl) Run(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C { //restic这东西, 只能串行执行, 不然其他任务会炸, 难, 总不能加个锁吧?
		now := time.Now()
		for _, bctx := range b.bctxList {
			if bctx.NextRunTime.After(now) {
				continue
			}
			b.runJob(ctx, bctx.Item)
			bctx.NextRunTime, _ = b.calcNextRunTime(bctx.Item.Expr) //重新计算下下次运行时间
		}
	}
	return nil
}

func (b *bkImpl) calcNextRunTime(expr string) (time.Time, error) {
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sc, err := p.Parse(expr)
	if err != nil {
		return time.Time{}, err
	}
	next := sc.Next(time.Now())
	return next, nil
}

func (b *bkImpl) runJob(ctx context.Context, item *backupItem) {
	ctx = trace.WithTraceId(ctx, fmt.Sprintf("BK:%s", item.Name))
	defer func() {
		if r := recover(); r != nil {
			logutil.GetLogger(ctx).Error("run item backup panic", zap.String("name", item.Name), zap.String("path", item.Path), zap.Any("expr", item.Expr),
				zap.Any("pre_run", item.PreRun), zap.Any("post_run", item.AfterRun), zap.Any("stack", string(debug.Stack())))
		}
	}()
	logutil.GetLogger(ctx).Info("start running backup",
		zap.String("name", item.Name),
		zap.String("path", item.Path),
		zap.String("expr", item.Expr),
		zap.Strings("pre_run", item.PreRun),
		zap.Strings("post_run", item.AfterRun),
	)
	start := time.Now()
	err := b.runItemBackup(ctx, item)
	end := time.Now()
	b.runNotify(ctx, item, start, end, err)
	b.runForget(ctx, item)
	logger := logutil.GetLogger(ctx).With(zap.String("name", item.Name), zap.String("path", item.Path), zap.Duration("cost", end.Sub(start)))
	if err != nil {
		logger.Error("backup item failed", zap.Error(err))
		return
	}
	logutil.GetLogger(ctx).Info("backup item succ")
}

func (b *bkImpl) runNotify(ctx context.Context, item *backupItem, start, end time.Time, err error) {
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

func (b *bkImpl) runForget(ctx context.Context, item *backupItem) {
	if b.c.keepRule.Last == 0 && b.c.keepRule.Daily == 0 && b.c.keepRule.Weekly == 0 &&
		b.c.keepRule.Monthly == 0 && b.c.keepRule.Yearly == 0 {
		return
	}
	if err := b.c.resitcer.Forget(ctx, b.c.keepRule.Last, b.c.keepRule.Daily, b.c.keepRule.Weekly,
		b.c.keepRule.Monthly, b.c.keepRule.Yearly); err != nil {
		logutil.GetLogger(ctx).Error("forget old backups failed",
			zap.String("name", item.Name), zap.Error(err))
	}
}

func (b *bkImpl) runItemBackup(ctx context.Context, item *backupItem) (err error) {
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

func (b *bkImpl) doBackup(ctx context.Context, item *backupItem) error {
	if err := b.c.resitcer.Backup(ctx, item.Path); err != nil {
		return fmt.Errorf("backup failed, path:%s, err:%w", item.Path, err)
	}
	return nil
}

func (b *bkImpl) runCmds(ctx context.Context, item *backupItem, cmds []string) error {
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

func (b *bkImpl) runCmd(ctx context.Context, workdir string, cmdstr string) error {
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
