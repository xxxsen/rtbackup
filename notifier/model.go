package notifier

type Notification struct {
	Title     string
	Path      string
	Start     int64
	End       int64
	IsSuccess bool
	Errmsg    string
}
