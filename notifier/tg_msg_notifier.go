package notifier

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

const (
	defaultTgTemplate = `
<b>{{.Title}}</b>
<b>Path</b>: {{.Path}}
<b>Start</b>: {{.Start | TsPrinter}}
<b>End</b>: {{.End | TsPrinter}}
<b>Result</b>: {{.IsSuccess | ResultPrinter}}	
{{- if not .IsSuccess}}
<b>ErrMsg</b>: {{.Errmsg}}
{{- end}}
`
)

type tgNotifier struct {
	host string
	user string
	pwd  string
	tplt *template.Template
}

func NewTGNotifier(host string, user string, pwd string) (INotifier, error) {
	if len(user) == 0 || len(pwd) == 0 || len(host) == 0 {
		return nil, fmt.Errorf("invalid params")
	}

	tplt, err := template.New("tg").Funcs(template.FuncMap{
		"TsPrinter": func(t int64) string {
			return time.UnixMilli(t).Format("2006-01-02 15:04:05")
		},
		"ResultPrinter": func(isSuccess bool) string {
			if isSuccess {
				return "Success"
			}
			return "Failed"
		},
	}).Parse(defaultTgTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse template failed, err:%w", err)
	}

	return &tgNotifier{
		host: host,
		user: user,
		pwd:  pwd,
		tplt: tplt,
	}, nil
}

func (n *tgNotifier) Name() string {
	return "tg"
}

func (n *tgNotifier) Notify(ctx context.Context, nt *Notification) error {
	buf := bytes.Buffer{}
	err := n.tplt.Execute(&buf, nt)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.host, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(n.user, n.pwd)
	req.Header.Add("x-message-type", "tg_raw_html")
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code:%d", rsp.StatusCode)
	}
	return nil
}
