// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"lunex/internal/runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func httpHeadersToObj(h http.Header) *runtime.Value {
	obj := make(map[string]*runtime.Value)
	for k, vals := range h {
		if len(vals) > 0 {
			obj[strings.ToLower(k)] = runtime.StringVal(strings.Join(vals, ", "))
		}
	}
	return runtime.ObjectVal(obj)
}

func httpBuildResponse(w http.ResponseWriter, status int, body string, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	fmt.Fprint(w, body)
}

func httpRequestObj(r *http.Request, body string) *runtime.Value {
	queryObj := make(map[string]*runtime.Value)
	for k, vals := range r.URL.Query() {
		if len(vals) > 0 {
			queryObj[k] = runtime.StringVal(vals[0])
		}
	}
	paramsObj := make(map[string]*runtime.Value)
	obj := runtime.ObjectVal(map[string]*runtime.Value{
		"method":  runtime.StringVal(r.Method),
		"url":     runtime.StringVal(r.URL.String()),
		"path":    runtime.StringVal(r.URL.Path),
		"query":   runtime.ObjectVal(queryObj),
		"params":  runtime.ObjectVal(paramsObj),
		"headers": httpHeadersToObj(r.Header),
		"body":    runtime.StringVal(body),
		"ip":      runtime.StringVal(r.RemoteAddr),
		"host":    runtime.StringVal(r.Host),
	})
	return obj
}

type httpServer struct {
	handler func(req *runtime.Value, res *runtime.Value) (*runtime.Value, error)
	mu      sync.Mutex
	routes  []httpRoute
}

type httpRoute struct {
	method  string
	pattern string
	handler *runtime.Value
}

func (s *httpServer) match(method, path string) (*runtime.Value, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.routes {
		if r.method != "" && r.method != method {
			continue
		}
		params, ok := matchRoute(r.pattern, path)
		if ok {
			return r.handler, params
		}
	}
	return nil, nil
}

func matchRoute(pattern, path string) (map[string]string, bool) {
	if pattern == path {
		return map[string]string{}, true
	}
	pParts := strings.Split(strings.Trim(pattern, "/"), "/")
	uParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pParts) != len(uParts) {
		if len(pParts) == 0 || pParts[len(pParts)-1] != "*" {
			return nil, false
		}
	}
	params := make(map[string]string)
	for i, pp := range pParts {
		if pp == "*" {
			return params, true
		}
		if i >= len(uParts) {
			return nil, false
		}
		up := uParts[i]
		if strings.HasPrefix(pp, ":") {
			params[pp[1:]] = up
		} else if pp != up {
			return nil, false
		}
	}
	return params, true
}

func httpServerVal(s *httpServer) *runtime.Value {
	obj := runtime.ObjectVal(map[string]*runtime.Value{
		"listen": runtime.FuncVal(&runtime.Function{Name: "listen", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			port := 3000
			host := "0.0.0.0"
			var onStart *runtime.Value
			if len(args) > 0 {
				port = int(args[0].ToNumber())
			}
			if len(args) > 1 {
				if args[1].Tag == runtime.TypeString {
					host = args[1].StrVal
				} else if args[1].Tag == runtime.TypeFunction {
					onStart = args[1]
				}
			}
			if len(args) > 2 && args[2].Tag == runtime.TypeFunction {
				onStart = args[2]
			}
			addr := fmt.Sprintf("%s:%d", host, port)
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return runtime.Null, err
			}
			if onStart != nil && runtime.CallFunction != nil {
				runtime.CallFunction(onStart, []*runtime.Value{runtime.NumberVal(float64(port))})
			}
			runtime.KeepAliveAdd()
			go func() {
				defer runtime.KeepAliveDone()
				mux := http.NewServeMux()
				mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					bodyBytes, _ := io.ReadAll(r.Body)
					body := string(bodyBytes)
					reqObj := httpRequestObj(r, body)
					var routeHandler *runtime.Value
					var params map[string]string
					s.mu.Lock()
					for _, route := range s.routes {
						if route.method != "" && route.method != r.Method {
							continue
						}
						p, ok := matchRoute(route.pattern, r.URL.Path)
						if ok {
							routeHandler = route.handler
							params = p
							break
						}
					}
					s.mu.Unlock()
					if params != nil {
						paramsObj := make(map[string]*runtime.Value)
						for k, v := range params {
							paramsObj[k] = runtime.StringVal(v)
						}
						reqObj.ObjVal["params"] = runtime.ObjectVal(paramsObj)
					}
					sent := false
					resObj := runtime.ObjectVal(map[string]*runtime.Value{
						"json": runtime.FuncVal(&runtime.Function{Name: "json", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
							if sent {
								return runtime.Undefined, nil
							}
							sent = true
							status := 200
							if len(a) > 1 {
								status = int(a[1].ToNumber())
							}
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(status)
							data := ""
							if len(a) > 0 {
								data = valueToJSON(a[0])
							}
							fmt.Fprint(w, data)
							return runtime.Undefined, nil
						}}),
						"send": runtime.FuncVal(&runtime.Function{Name: "send", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
							if sent {
								return runtime.Undefined, nil
							}
							sent = true
							status := 200
							body := ""
							ct := "text/plain"
							if len(a) > 0 {
								if a[0].Tag == runtime.TypeObject || a[0].Tag == runtime.TypeArray {
									body = valueToJSON(a[0])
									ct = "application/json"
								} else {
									body = a[0].ToString()
								}
							}
							if len(a) > 1 {
								status = int(a[1].ToNumber())
							}
							httpBuildResponse(w, status, body, ct)
							return runtime.Undefined, nil
						}}),
						"text": runtime.FuncVal(&runtime.Function{Name: "text", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
							if sent {
								return runtime.Undefined, nil
							}
							sent = true
							status := 200
							body := ""
							if len(a) > 0 {
								body = a[0].ToString()
							}
							if len(a) > 1 {
								status = int(a[1].ToNumber())
							}
							httpBuildResponse(w, status, body, "text/plain; charset=utf-8")
							return runtime.Undefined, nil
						}}),
						"html": runtime.FuncVal(&runtime.Function{Name: "html", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
							if sent {
								return runtime.Undefined, nil
							}
							sent = true
							status := 200
							body := ""
							if len(a) > 0 {
								body = a[0].ToString()
							}
							if len(a) > 1 {
								status = int(a[1].ToNumber())
							}
							httpBuildResponse(w, status, body, "text/html; charset=utf-8")
							return runtime.Undefined, nil
						}}),
						"redirect": runtime.FuncVal(&runtime.Function{Name: "redirect", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
							if sent {
								return runtime.Undefined, nil
							}
							sent = true
							url := "/"
							status := 302
							if len(a) > 0 {
								url = a[0].ToString()
							}
							if len(a) > 1 {
								status = int(a[1].ToNumber())
							}
							http.Redirect(w, r, url, status)
							return runtime.Undefined, nil
						}}),
						"status": runtime.FuncVal(&runtime.Function{Name: "status", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
							if len(a) > 0 && !sent {
								w.WriteHeader(int(a[0].ToNumber()))
							}
							return runtime.Undefined, nil
						}}),
						"setHeader": runtime.FuncVal(&runtime.Function{Name: "setHeader", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
							if len(a) >= 2 && !sent {
								w.Header().Set(a[0].ToString(), a[1].ToString())
							}
							return runtime.Undefined, nil
						}}),
						"end": runtime.FuncVal(&runtime.Function{Name: "end", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
							if !sent {
								sent = true
								w.WriteHeader(200)
							}
							return runtime.Undefined, nil
						}}),
					})
					if routeHandler != nil && runtime.CallFunction != nil {
						res, err := runtime.CallFunction(routeHandler, []*runtime.Value{reqObj, resObj})
						if err == nil && res != nil && !sent {
							if res.Tag == runtime.TypeString {
								httpBuildResponse(w, 200, res.StrVal, "text/plain")
							} else if res.Tag == runtime.TypeObject || res.Tag == runtime.TypeArray {
								httpBuildResponse(w, 200, valueToJSON(res), "application/json")
							}
						}
					} else if s.handler != nil {
						res, err := s.handler(reqObj, resObj)
						if err == nil && res != nil && !sent {
							if res.Tag == runtime.TypeString {
								httpBuildResponse(w, 200, res.StrVal, "text/plain")
							} else if res.Tag == runtime.TypeObject || res.Tag == runtime.TypeArray {
								httpBuildResponse(w, 200, valueToJSON(res), "application/json")
							}
						}
					} else if !sent {
						httpBuildResponse(w, 404, `{"error":"not found"}`, "application/json")
					}
				})
				http.Serve(ln, mux)
			}()
			return runtime.ObjectVal(map[string]*runtime.Value{
				"port":  runtime.NumberVal(float64(port)),
				"close": runtime.FuncVal(&runtime.Function{Name: "close", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) { ln.Close(); return runtime.Undefined, nil }}),
			}), nil
		}}),

		"get": runtime.FuncVal(&runtime.Function{Name: "get", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := ""
			if len(args) > 0 { pattern = args[0].ToString() }
			var handler *runtime.Value
			if len(args) > 1 { handler = args[1] }
			s.routes = append(s.routes, httpRoute{"GET", pattern, handler})
			s.mu.Unlock()
			return httpServerVal(s), nil
		}}),
		"post": runtime.FuncVal(&runtime.Function{Name: "post", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := ""
			if len(args) > 0 { pattern = args[0].ToString() }
			var handler *runtime.Value
			if len(args) > 1 { handler = args[1] }
			s.routes = append(s.routes, httpRoute{"POST", pattern, handler})
			s.mu.Unlock()
			return httpServerVal(s), nil
		}}),
		"put": runtime.FuncVal(&runtime.Function{Name: "put", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := ""
			if len(args) > 0 { pattern = args[0].ToString() }
			var handler *runtime.Value
			if len(args) > 1 { handler = args[1] }
			s.routes = append(s.routes, httpRoute{"PUT", pattern, handler})
			s.mu.Unlock()
			return httpServerVal(s), nil
		}}),
		"patch": runtime.FuncVal(&runtime.Function{Name: "patch", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := ""
			if len(args) > 0 { pattern = args[0].ToString() }
			var handler *runtime.Value
			if len(args) > 1 { handler = args[1] }
			s.routes = append(s.routes, httpRoute{"PATCH", pattern, handler})
			s.mu.Unlock()
			return httpServerVal(s), nil
		}}),
		"delete": runtime.FuncVal(&runtime.Function{Name: "delete", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := ""
			if len(args) > 0 { pattern = args[0].ToString() }
			var handler *runtime.Value
			if len(args) > 1 { handler = args[1] }
			s.routes = append(s.routes, httpRoute{"DELETE", pattern, handler})
			s.mu.Unlock()
			return httpServerVal(s), nil
		}}),
		"all": runtime.FuncVal(&runtime.Function{Name: "all", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := ""
			if len(args) > 0 { pattern = args[0].ToString() }
			var handler *runtime.Value
			if len(args) > 1 { handler = args[1] }
			s.routes = append(s.routes, httpRoute{"", pattern, handler})
			s.mu.Unlock()
			return httpServerVal(s), nil
		}}),
		"use": runtime.FuncVal(&runtime.Function{Name: "use", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s.mu.Lock()
			pattern := "/"
			var handler *runtime.Value
			if len(args) == 1 { handler = args[0] } else if len(args) >= 2 { pattern = args[0].ToString(); handler = args[1] }
			if handler != nil { s.routes = append([]httpRoute{{"", pattern, handler}}, s.routes...) }
			s.mu.Unlock()
			return httpServerVal(s), nil
		}}),
	})
	return obj
}

func HttpModule() *runtime.Value {
	client := &http.Client{Timeout: 30 * time.Second}

	doRequest := func(method, url string, opts *runtime.Value) (*runtime.Value, error) {
		var body io.Reader
		headers := map[string]string{"Content-Type": "application/json"}
		if opts != nil && opts.Tag == runtime.TypeObject {
			if b, ok := opts.ObjVal["body"]; ok && b != nil {
				var bodyStr string
				if b.Tag == runtime.TypeObject || b.Tag == runtime.TypeArray {
					bodyStr = valueToJSON(b)
				} else {
					bodyStr = b.ToString()
				}
				body = bytes.NewBufferString(bodyStr)
			}
			if h, ok := opts.ObjVal["headers"]; ok && h != nil && h.Tag == runtime.TypeObject {
				for k, v := range h.ObjVal {
					headers[k] = v.ToString()
				}
			}
			if t, ok := opts.ObjVal["timeout"]; ok && t != nil {
				client.Timeout = time.Duration(t.ToNumber()) * time.Millisecond
			}
		}
		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return runtime.ObjectVal(map[string]*runtime.Value{"ok": runtime.False, "error": runtime.StringVal(err.Error())}), nil
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			return runtime.ObjectVal(map[string]*runtime.Value{"ok": runtime.False, "error": runtime.StringVal(err.Error())}), nil
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		bodyStr := string(respBody)
		ct := resp.Header.Get("Content-Type")
		var parsedBody *runtime.Value
		if strings.Contains(ct, "application/json") {
			parsed, err := parseJSON(bodyStr)
			if err == nil {
				parsedBody = parsed
			} else {
				parsedBody = runtime.StringVal(bodyStr)
			}
		} else {
			parsedBody = runtime.StringVal(bodyStr)
		}
		return runtime.ObjectVal(map[string]*runtime.Value{
			"ok":      runtime.BoolVal(resp.StatusCode >= 200 && resp.StatusCode < 300),
			"status":  runtime.NumberVal(float64(resp.StatusCode)),
			"headers": httpHeadersToObj(resp.Header),
			"body":    parsedBody,
			"text":    runtime.StringVal(bodyStr),
			"json": runtime.FuncVal(&runtime.Function{Name: "json", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				v, err := parseJSON(bodyStr)
				if err != nil { return runtime.Null, nil }
				return v, nil
			}}),
		}), nil
	}

	return runtime.ObjectVal(map[string]*runtime.Value{
		"request": runtime.FuncVal(&runtime.Function{Name: "request", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 { return runtime.Null, fmt.Errorf("http.request: method and url required") }
			var opts *runtime.Value
			if len(args) > 2 { opts = args[2] }
			return doRequest(args[0].ToString(), args[1].ToString(), opts)
		}}),

		"get": runtime.FuncVal(&runtime.Function{Name: "get", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.Null, fmt.Errorf("url required") }
			var opts *runtime.Value
			if len(args) > 1 { opts = args[1] }
			return doRequest("GET", args[0].ToString(), opts)
		}}),
		"post": runtime.FuncVal(&runtime.Function{Name: "post", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.Null, fmt.Errorf("url required") }
			var opts *runtime.Value
			if len(args) > 1 { opts = args[1] }
			return doRequest("POST", args[0].ToString(), opts)
		}}),
		"put": runtime.FuncVal(&runtime.Function{Name: "put", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.Null, fmt.Errorf("url required") }
			var opts *runtime.Value
			if len(args) > 1 { opts = args[1] }
			return doRequest("PUT", args[0].ToString(), opts)
		}}),
		"patch": runtime.FuncVal(&runtime.Function{Name: "patch", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.Null, fmt.Errorf("url required") }
			var opts *runtime.Value
			if len(args) > 1 { opts = args[1] }
			return doRequest("PATCH", args[0].ToString(), opts)
		}}),
		"delete": runtime.FuncVal(&runtime.Function{Name: "delete", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.Null, fmt.Errorf("url required") }
			var opts *runtime.Value
			if len(args) > 1 { opts = args[1] }
			return doRequest("DELETE", args[0].ToString(), opts)
		}}),
		"head": runtime.FuncVal(&runtime.Function{Name: "head", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.Null, fmt.Errorf("url required") }
			return doRequest("HEAD", args[0].ToString(), nil)
		}}),

		"createServer": runtime.FuncVal(&runtime.Function{Name: "createServer", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s := &httpServer{}
			if len(args) > 0 && args[0].Tag == runtime.TypeFunction {
				fn := args[0]
				s.handler = func(req *runtime.Value, res *runtime.Value) (*runtime.Value, error) {
					if runtime.CallFunction != nil {
						return runtime.CallFunction(fn, []*runtime.Value{req, res})
					}
					return runtime.Undefined, nil
				}
			}
			return httpServerVal(s), nil
		}}),

		"listen": runtime.FuncVal(&runtime.Function{Name: "listen", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 { return runtime.Null, fmt.Errorf("server and port required") }
			server := args[0]
			if listen, ok := server.ObjVal["listen"]; ok {
				return runtime.CallFunction(listen, args[1:])
			}
			return runtime.Undefined, nil
		}}),

		"json": runtime.FuncVal(&runtime.Function{Name: "json", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 { return runtime.Undefined, nil }
			res := args[0]
			data := args[1]
			status := 200
			if len(args) > 2 { status = int(args[2].ToNumber()) }
			if jsonFn, ok := res.ObjVal["json"]; ok {
				return runtime.CallFunction(jsonFn, []*runtime.Value{data, runtime.NumberVal(float64(status))})
			}
			return runtime.Undefined, nil
		}}),

		"text": runtime.FuncVal(&runtime.Function{Name: "text", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 { return runtime.Undefined, nil }
			res := args[0]
			if textFn, ok := res.ObjVal["text"]; ok {
				return runtime.CallFunction(textFn, args[1:])
			}
			return runtime.Undefined, nil
		}}),

		"redirect": runtime.FuncVal(&runtime.Function{Name: "redirect", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 { return runtime.Undefined, nil }
			res := args[0]
			if redFn, ok := res.ObjVal["redirect"]; ok {
				return runtime.CallFunction(redFn, args[1:])
			}
			return runtime.Undefined, nil
		}}),

		"parseBody": runtime.FuncVal(&runtime.Function{Name: "parseBody", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.Null, nil }
			req := args[0]
			if body, ok := req.ObjVal["body"]; ok {
				if body.Tag == runtime.TypeString {
					v, err := parseJSON(body.StrVal)
					if err == nil { return v, nil }
					return body, nil
				}
				return body, nil
			}
			return runtime.Null, nil
		}}),

		"serveStatic": runtime.FuncVal(&runtime.Function{Name: "serveStatic", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			dir := "."
			if len(args) > 0 { dir = args[0].ToString() }
			return runtime.FuncVal(&runtime.Function{Name: "staticHandler", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				return runtime.ObjectVal(map[string]*runtime.Value{"staticDir": runtime.StringVal(dir)}), nil
			}}), nil
		}}),

		"cookie": runtime.FuncVal(&runtime.Function{Name: "cookie", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 { return runtime.Undefined, nil }
			return runtime.Undefined, nil
		}}),

		"parseURL": runtime.FuncVal(&runtime.Function{Name: "parseURL", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.ObjectVal(nil), nil }
			rawURL := args[0].ToString()
			protocol := ""
			host := ""
			path := rawURL
			query := ""
			if idx := strings.Index(rawURL, "://"); idx >= 0 {
				protocol = rawURL[:idx]
				rest := rawURL[idx+3:]
				if slash := strings.Index(rest, "/"); slash >= 0 {
					host = rest[:slash]
					path = rest[slash:]
				} else {
					host = rest
					path = "/"
				}
			}
			if idx := strings.Index(path, "?"); idx >= 0 {
				query = path[idx+1:]
				path = path[:idx]
			}
			queryObj := make(map[string]*runtime.Value)
			for _, part := range strings.Split(query, "&") {
				kv := strings.SplitN(part, "=", 2)
				if len(kv) == 2 {
					queryObj[kv[0]] = runtime.StringVal(kv[1])
				}
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"protocol": runtime.StringVal(protocol),
				"host":     runtime.StringVal(host),
				"path":     runtime.StringVal(path),
				"query":    runtime.ObjectVal(queryObj),
				"search":   runtime.StringVal(query),
			}), nil
		}}),

		"buildURL": runtime.FuncVal(&runtime.Function{Name: "buildURL", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.StringVal(""), nil }
			base := args[0].ToString()
			if len(args) > 1 && args[1].Tag == runtime.TypeObject {
				params := []string{}
				for k, v := range args[1].ObjVal {
					params = append(params, k+"="+v.ToString())
				}
				if len(params) > 0 {
					if strings.Contains(base, "?") {
						base += "&" + strings.Join(params, "&")
					} else {
						base += "?" + strings.Join(params, "&")
					}
				}
			}
			return runtime.StringVal(base), nil
		}}),

		"encode": runtime.FuncVal(&runtime.Function{Name: "encode", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.StringVal(""), nil }
			return runtime.StringVal(urlEncode(args[0].ToString())), nil
		}}),
		"decode": runtime.FuncVal(&runtime.Function{Name: "decode", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.StringVal(""), nil }
			return runtime.StringVal(urlDecode(args[0].ToString())), nil
		}}),

		"statusText": runtime.FuncVal(&runtime.Function{Name: "statusText", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 { return runtime.StringVal(""), nil }
			return runtime.StringVal(http.StatusText(int(args[0].ToNumber()))), nil
		}}),
	})
}

func urlEncode(s string) string {
	var out strings.Builder
	for _, c := range s {
		switch {
		case (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~':
			out.WriteRune(c)
		default:
			out.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return out.String()
}

func urlDecode(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			val, err := strconv.ParseInt(s[i+1:i+3], 16, 32)
			if err == nil {
				out.WriteRune(rune(val))
				i += 2
				continue
			}
		} else if s[i] == '+' {
			out.WriteByte(' ')
			continue
		}
		out.WriteByte(s[i])
	}
	return out.String()
}

func init() {
	_ = json.Marshal
}
