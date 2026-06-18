//go:build !js

// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package std

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"lunex/internal/runtime"
	"strings"
	"sync"
)

type wsConn struct {
	conn    net.Conn
	mu      sync.Mutex
	closed  bool
	onMsg   *runtime.Value
	onClose *runtime.Value
}

func (c *wsConn) send(msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("connection closed")
	}
	payload := []byte(msg)
	n := len(payload)
	var header []byte
	header = append(header, 0x81)
	if n < 126 {
		header = append(header, byte(n))
	} else if n < 65536 {
		header = append(header, 126, byte(n>>8), byte(n))
	} else {
		header = append(header, 127)
		for i := 7; i >= 0; i-- {
			header = append(header, byte(n>>(uint(i)*8)))
		}
	}
	_, err := c.conn.Write(append(header, payload...))
	return err
}

func (c *wsConn) close() {
	c.mu.Lock()
	if !c.closed {
		c.closed = true
		c.conn.Write([]byte{0x88, 0x00})
		c.conn.Close()
	}
	c.mu.Unlock()
}

func (c *wsConn) readLoop() {
	defer func() {
		c.close()
		if c.onClose != nil && runtime.CallFunction != nil {
			runtime.CallFunction(c.onClose, nil, nil)
		}
	}()
	buf := bufio.NewReader(c.conn)
	for {
		b0, err := buf.ReadByte()
		if err != nil {
			return
		}
		opcode := b0 & 0x0f
		if opcode == 0x8 {
			return
		}
		b1, err := buf.ReadByte()
		if err != nil {
			return
		}
		masked := (b1 & 0x80) != 0
		payloadLen := int(b1 & 0x7f)
		if payloadLen == 126 {
			hi, _ := buf.ReadByte()
			lo, _ := buf.ReadByte()
			payloadLen = int(hi)<<8 | int(lo)
		} else if payloadLen == 127 {
			var l int
			for i := 0; i < 8; i++ {
				b, _ := buf.ReadByte()
				l = l<<8 | int(b)
			}
			payloadLen = l
		}
		var mask [4]byte
		if masked {
			for i := 0; i < 4; i++ {
				mask[i], _ = buf.ReadByte()
			}
		}
		payload := make([]byte, payloadLen)
		for i := range payload {
			payload[i], err = buf.ReadByte()
			if err != nil {
				return
			}
			if masked {
				payload[i] ^= mask[i%4]
			}
		}
		if c.onMsg != nil && runtime.CallFunction != nil {
			runtime.CallFunction(c.onMsg, []*runtime.Value{runtime.StringVal(string(payload))}, nil)
		}
	}
}

func wsHandshake(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
	key := r.Header.Get("Sec-Websocket-Key")
	if key == "" {
		return nil, fmt.Errorf("missing Sec-WebSocket-Key")
	}
	accept := computeWSAccept(key)
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("hijacking not supported")
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	resp := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\n\r\n"
	buf.WriteString(resp)
	buf.Flush()
	return conn, nil
}

func computeWSAccept(key string) string {
	const magic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key + magic))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func wsConnValue(c *wsConn) *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"send": runtime.FuncVal(&runtime.Function{Name: "send", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.Undefined, nil }
			msg := args[0].ToString()
			if args[0].Tag == runtime.TypeObject || args[0].Tag == runtime.TypeArray {
				msg = valueToJSON(args[0])
			}
			c.send(msg)
			return runtime.Undefined, nil
		}}),
		"close": runtime.FuncVal(&runtime.Function{Name: "close", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			c.close()
			return runtime.Undefined, nil
		}}),
		"onMessage": runtime.FuncVal(&runtime.Function{Name: "onMessage", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 && args[0].Tag == runtime.TypeFunction { c.onMsg = args[0] }
			return runtime.Undefined, nil
		}}),
		"onClose": runtime.FuncVal(&runtime.Function{Name: "onClose", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 && args[0].Tag == runtime.TypeFunction { c.onClose = args[0] }
			return runtime.Undefined, nil
		}}),
		"isClosed": runtime.FuncVal(&runtime.Function{Name: "isClosed", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.BoolVal(c.closed), nil
		}}),
	})
}

func WsModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"createServer": runtime.FuncVal(&runtime.Function{Name: "createServer", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			port := 8080
			if len(args) > 0 { port = int(args[0].ToNumber()) }
			var connHandler *runtime.Value
			if len(args) >= 3 { connHandler = args[2] } else if len(args) == 2 && args[1].Tag == runtime.TypeFunction { connHandler = args[1] }

			var clients []*wsConn
			var clientsMu sync.Mutex

			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
					http.Error(w, "WebSocket only", 400)
					return
				}
				conn, err := wsHandshake(w, r)
				if err != nil {
					return
				}
				c := &wsConn{conn: conn}
				connVal := wsConnValue(c)
				clientsMu.Lock()
				clients = append(clients, c)
				clientsMu.Unlock()
				c.onClose = runtime.FuncVal(&runtime.Function{Name: "_onClose", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					clientsMu.Lock()
					for i, cl := range clients {
						if cl == c {
							clients = append(clients[:i], clients[i+1:]...)
							break
						}
					}
					clientsMu.Unlock()
					return runtime.Undefined, nil
				}})
				if connHandler != nil && runtime.CallFunction != nil {
					runtime.CallFunction(connHandler, []*runtime.Value{connVal}, nil)
				}
				go c.readLoop()
			})

			ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
			if err != nil {
				return runtime.Null, err
			}
			runtime.KeepAliveAdd()
			go func() {
				defer runtime.KeepAliveDone()
				http.Serve(ln, mux)
			}()

			return runtime.ObjectVal(map[string]*runtime.Value{
				"port": runtime.NumberVal(float64(port)),
				"broadcast": runtime.FuncVal(&runtime.Function{Name: "broadcast", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					if len(a) == 0 { return runtime.Undefined, nil }
					msg := a[0].ToString()
					if a[0].Tag == runtime.TypeObject || a[0].Tag == runtime.TypeArray { msg = valueToJSON(a[0]) }
					clientsMu.Lock()
					snap := make([]*wsConn, len(clients))
					copy(snap, clients)
					clientsMu.Unlock()
					for _, cl := range snap { cl.send(msg) }
					return runtime.Undefined, nil
				}}),
				"clientCount": runtime.FuncVal(&runtime.Function{Name: "clientCount", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					clientsMu.Lock()
					n := len(clients)
					clientsMu.Unlock()
					return runtime.NumberVal(float64(n)), nil
				}}),
				"close": runtime.FuncVal(&runtime.Function{Name: "close", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					ln.Close()
					return runtime.Undefined, nil
				}}),
			}), nil
		}}),

		"send": runtime.FuncVal(&runtime.Function{Name: "send", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 { return runtime.Undefined, nil }
			if sendFn, ok := args[0].ObjVal["send"]; ok {
				runtime.CallFunction(sendFn, []*runtime.Value{args[1]}, nil)
			}
			return runtime.Undefined, nil
		}}),

		"onMessage": runtime.FuncVal(&runtime.Function{Name: "onMessage", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 { return runtime.Undefined, nil }
			if fn, ok := args[0].ObjVal["onMessage"]; ok {
				runtime.CallFunction(fn, []*runtime.Value{args[1]}, nil)
			}
			return runtime.Undefined, nil
		}}),

		"onClose": runtime.FuncVal(&runtime.Function{Name: "onClose", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 { return runtime.Undefined, nil }
			if fn, ok := args[0].ObjVal["onClose"]; ok {
				runtime.CallFunction(fn, []*runtime.Value{args[1]}, nil)
			}
			return runtime.Undefined, nil
		}}),

		"closeServer": runtime.FuncVal(&runtime.Function{Name: "closeServer", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				if closeFn, ok := args[0].ObjVal["close"]; ok {
					runtime.CallFunction(closeFn, nil, nil)
				}
			}
			return runtime.Undefined, nil
		}}),

		"connect": runtime.FuncVal(&runtime.Function{Name: "connect", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.Null, fmt.Errorf("url required") }
			return runtime.ObjectVal(map[string]*runtime.Value{
				"send": runtime.FuncVal(&runtime.Function{Name: "send", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) { return runtime.Undefined, nil }}),
				"close": runtime.FuncVal(&runtime.Function{Name: "close", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) { return runtime.Undefined, nil }}),
			}), nil
		}}),

		"closeClient": runtime.FuncVal(&runtime.Function{Name: "closeClient", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				if closeFn, ok := args[0].ObjVal["close"]; ok {
					runtime.CallFunction(closeFn, nil, nil)
				}
			}
			return runtime.Undefined, nil
		}}),
	})
}
