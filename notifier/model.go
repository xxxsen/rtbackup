package notifier

type Notification struct {
	Title     string `json:"title"`
	Path      string `json:"path"`
	Start     int64  `json:"start"`
	End       int64  `json:"end"`
	IsSuccess bool   `json:"is_success"`
}
