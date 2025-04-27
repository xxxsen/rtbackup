package notifier

import (
	"context"
)

var (
	Nop = NewNopNotifier()
)

type nopNotifier struct{}

func (n *nopNotifier) Name() string {
	return "nop"
}

func (n *nopNotifier) Notify(ctx context.Context, msg *Notification) error {
	return nil
}

func NewNopNotifier() INotifier {
	return &nopNotifier{}
}
