// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"fmt"
	"net"
	"net/smtp"
	"lunex/internal/runtime"
	"strings"
	"sync/atomic"
)

var mailCounter uint64

type mailerConfig struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func mailerVal(cfg mailerConfig) *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"send": runtime.FuncVal(&runtime.Function{Name: "send", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.False, fmt.Errorf("send: options required")
			}
			opts := args[0].ObjVal
			to := ""
			if v, ok := opts["to"]; ok { to = v.ToString() }
			subject := ""
			if v, ok := opts["subject"]; ok { subject = v.ToString() }
			body := ""
			isHTML := false
			if v, ok := opts["html"]; ok { body = v.ToString(); isHTML = true } else if v, ok := opts["text"]; ok { body = v.ToString() }
			from := cfg.from
			if v, ok := opts["from"]; ok { from = v.ToString() }

			addr := fmt.Sprintf("%s:%d", cfg.host, cfg.port)

			var auth smtp.Auth
			if cfg.user != "" {
				auth = smtp.PlainAuth("", cfg.user, cfg.password, cfg.host)
			}

			ct := "text/plain"
			if isHTML { ct = "text/html" }

			headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: %s; charset=UTF-8\r\nMIME-Version: 1.0\r\n\r\n", from, to, subject, ct)
			msg := headers + body

			toAddrs := strings.Split(to, ",")
			for i, a := range toAddrs {
				toAddrs[i] = strings.TrimSpace(a)
			}

			id := atomic.AddUint64(&mailCounter, 1)
			msgID := fmt.Sprintf("<lunex-%d@%s>", id, cfg.host)

			if cfg.host == "" || cfg.host == "mock" {
				return runtime.ObjectVal(map[string]*runtime.Value{
					"messageId": runtime.StringVal(msgID),
					"accepted":  runtime.ArrayVal([]*runtime.Value{runtime.StringVal(to)}),
					"mock":      runtime.True,
				}), nil
			}

			err := smtp.SendMail(addr, auth, from, toAddrs, []byte(msg))
			if err != nil {
				netErr, isNet := err.(net.Error)
				if isNet && netErr.Timeout() {
					return runtime.ObjectVal(map[string]*runtime.Value{
						"error":   runtime.StringVal("connection timeout"),
						"timeout": runtime.True,
					}), nil
				}
				return runtime.ObjectVal(map[string]*runtime.Value{
					"error": runtime.StringVal(err.Error()),
				}), nil
			}

			return runtime.ObjectVal(map[string]*runtime.Value{
				"messageId": runtime.StringVal(msgID),
				"accepted":  runtime.ArrayVal([]*runtime.Value{runtime.StringVal(to)}),
			}), nil
		}}),
		"verify": runtime.FuncVal(&runtime.Function{Name: "verify", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if cfg.host == "" || cfg.host == "mock" {
				return runtime.True, nil
			}
			addr := fmt.Sprintf("%s:%d", cfg.host, cfg.port)
			c, err := smtp.Dial(addr)
			if err != nil {
				return runtime.False, nil
			}
			c.Quit()
			return runtime.True, nil
		}}),
		"config": runtime.ObjectVal(map[string]*runtime.Value{
			"host": runtime.StringVal(cfg.host),
			"port": runtime.NumberVal(float64(cfg.port)),
			"user": runtime.StringVal(cfg.user),
			"from": runtime.StringVal(cfg.from),
		}),
	})
}

func MailModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"createTransport": runtime.FuncVal(&runtime.Function{Name: "createTransport", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			cfg := mailerConfig{host: "mock", port: 587}
			if len(args) > 0 && args[0].Tag == runtime.TypeObject {
				opts := args[0].ObjVal
				if v, ok := opts["host"]; ok { cfg.host = v.ToString() }
				if v, ok := opts["port"]; ok { cfg.port = int(v.ToNumber()) }
				if v, ok := opts["user"]; ok { cfg.user = v.ToString() }
				if v, ok := opts["username"]; ok { cfg.user = v.ToString() }
				if v, ok := opts["password"]; ok { cfg.password = v.ToString() }
				if v, ok := opts["pass"]; ok { cfg.password = v.ToString() }
				if v, ok := opts["from"]; ok { cfg.from = v.ToString() }
				if v, ok := opts["sender"]; ok { cfg.from = v.ToString() }
				if v, ok := opts["service"]; ok {
					switch strings.ToLower(v.ToString()) {
					case "gmail":
						cfg.host = "smtp.gmail.com"
						cfg.port = 587
					case "sendgrid":
						cfg.host = "smtp.sendgrid.net"
						cfg.port = 587
					case "mailgun":
						cfg.host = "smtp.mailgun.org"
						cfg.port = 587
					case "ses", "amazon":
						cfg.host = "email-smtp.us-east-1.amazonaws.com"
						cfg.port = 587
					}
				}
			}
			return mailerVal(cfg), nil
		}}),

		"send": runtime.FuncVal(&runtime.Function{Name: "send", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 { return runtime.False, nil }
			transport := args[0]
			if sendFn, ok := transport.ObjVal["send"]; ok {
				return runtime.CallFunction(sendFn, []*runtime.Value{args[1]}, nil)
			}
			return runtime.False, nil
		}}),

		"createMock": runtime.FuncVal(&runtime.Function{Name: "createMock", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			cfg := mailerConfig{host: "mock", port: 587, from: "noreply@mock.test"}
			return mailerVal(cfg), nil
		}}),
	})
}
