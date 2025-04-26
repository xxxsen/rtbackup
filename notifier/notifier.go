package notifier

import (
	"context"
)

type INotifier interface {
	Name() string
	Notify(ctx context.Context, msg *Notification) error
}
