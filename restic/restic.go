package restic

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/xxxsen/common/logutil"
	"go.uber.org/zap"
)

type IResitc interface {
	Backup(ctx context.Context, location string) error
	Unlock(ctx context.Context) error
	Forget(ctx context.Context, last, daily, weekly, monthly, yearly int) error
}

type resticImpl struct {
	bin string
	c   *config
}

func New(opts ...Option) (IResitc, error) {
	p, err := exec.LookPath("restic")
	if err != nil {
		return nil, fmt.Errorf("lookup restic binary failed, err:%w", err)
	}
	r := &resticImpl{
		bin: p,
	}

	c := &config{}
	for _, opt := range opts {
		opt(c)
	}

	r.c = c
	return r, nil
}

func (r *resticImpl) runCmd(ctx context.Context, args ...string) error {
	args = append(args, "--cache-dir", r.c.cacheDir, "--cleanup-cache")
	cmd := exec.CommandContext(ctx, r.bin, args...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("RESTIC_REPOSITORY=%s", r.c.repo))
	if r.c.pwd != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RESTIC_PASSWORD=%s", r.c.pwd))
	}
	cmd.Stdout = nil
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr

	logutil.GetLogger(ctx).Debug("start exec cmd", zap.String("cmd", r.bin), zap.Strings("args", args))
	start := time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running restic command failed, err:%w, errmsg:%s", err, stderr.String())
	}
	logutil.GetLogger(ctx).Debug("exec cmd finished", zap.String("cmd", r.bin), zap.Strings("args", args), zap.Duration("cost", time.Since(start)))
	return nil
}

func (r *resticImpl) Backup(ctx context.Context, location string) error {
	return r.runCmd(ctx, "backup", location)
}

func (r *resticImpl) Unlock(ctx context.Context) error {
	return r.runCmd(ctx, "unlock")
}

func (r *resticImpl) Forget(ctx context.Context, last, daily, weekly, monthly, yearly int) error {
	args := []string{"forget", "--prune"}
	if last > 0 {
		args = append(args, fmt.Sprintf("--keep-last=%d", last))
	}
	if daily > 0 {
		args = append(args, fmt.Sprintf("--keep-daily=%d", daily))
	}
	if weekly > 0 {
		args = append(args, fmt.Sprintf("--keep-weekly=%d", weekly))
	}
	if monthly > 0 {
		args = append(args, fmt.Sprintf("--keep-monthly=%d", monthly))
	}
	if yearly > 0 {
		args = append(args, fmt.Sprintf("--keep-yearly=%d", yearly))
	}

	return r.runCmd(ctx, args...)
}
