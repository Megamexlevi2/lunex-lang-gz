// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
        "bytes"
        "encoding/json"
        "fmt"
        "io"
        "math"
        "net/http"
        "lunex/internal/runtime"
        "os"
        "strings"
        "time"
)

type aiClient struct {
        apiKey   string
        baseURL  string
        model    string
        provider string
        timeout  time.Duration
}

func (c *aiClient) chat(messages []*runtime.Value, opts *runtime.Value) (*runtime.Value, error) {
        msgs := make([]map[string]interface{}, 0, len(messages))
        for _, m := range messages {
                if m == nil || m.Tag != runtime.TypeObject {
                        continue
                }
                role := "user"
                content := ""
                if r, ok := m.ObjVal["role"]; ok {
                        role = r.ToString()
                }
                if ct, ok := m.ObjVal["content"]; ok {
                        content = ct.ToString()
                }
                msgs = append(msgs, map[string]interface{}{"role": role, "content": content})
        }

        model := c.model
        if model == "" {
                model = "gpt-3.5-turbo"
        }
        maxTokens := 1024
        temperature := 0.7

        if opts != nil && opts.Tag == runtime.TypeObject {
                if v, ok := opts.ObjVal["model"]; ok {
                        model = v.ToString()
                }
                if v, ok := opts.ObjVal["maxTokens"]; ok {
                        maxTokens = int(v.ToNumber())
                }
                if v, ok := opts.ObjVal["max_tokens"]; ok {
                        maxTokens = int(v.ToNumber())
                }
                if v, ok := opts.ObjVal["temperature"]; ok {
                        temperature = v.ToNumber()
                }
        }

        payload := map[string]interface{}{
                "model":       model,
                "messages":    msgs,
                "max_tokens":  maxTokens,
                "temperature": temperature,
        }

        body, err := json.Marshal(payload)
        if err != nil {
                return runtime.Null, err
        }

        baseURL := c.baseURL
        if baseURL == "" {
                baseURL = "https://api.openai.com"
        }

        req, err := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(body))
        if err != nil {
                return runtime.Null, err
        }
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("Authorization", "Bearer "+c.apiKey)

        client := &http.Client{Timeout: c.timeout}
        resp, err := client.Do(req)
        if err != nil {
                return runtime.ObjectVal(map[string]*runtime.Value{
                        "error": runtime.StringVal(err.Error()),
                }), nil
        }
        defer resp.Body.Close()
        respBody, _ := io.ReadAll(resp.Body)

        var result map[string]interface{}
        if err := json.Unmarshal(respBody, &result); err != nil {
                return runtime.StringVal(string(respBody)), nil
        }

        choices, ok := result["choices"].([]interface{})
        if !ok || len(choices) == 0 {
                v, _ := parseJSON(string(respBody))
                return v, nil
        }

        choice, _ := choices[0].(map[string]interface{})
        msg, _ := choice["message"].(map[string]interface{})
        content := ""
        if msg != nil {
                content = fmt.Sprintf("%v", msg["content"])
        }

        return runtime.ObjectVal(map[string]*runtime.Value{
                "message": runtime.ObjectVal(map[string]*runtime.Value{
                        "role":    runtime.StringVal("assistant"),
                        "content": runtime.StringVal(content),
                }),
                "content": runtime.StringVal(content),
                "model":   runtime.StringVal(model),
                "usage": func() *runtime.Value {
                        if u, ok := result["usage"].(map[string]interface{}); ok {
                                return jsonToValue(u)
                        }
                        return runtime.ObjectVal(nil)
                }(),
        }), nil
}

func (c *aiClient) complete(prompt string, opts *runtime.Value) (*runtime.Value, error) {
        msgs := []*runtime.Value{
                runtime.ObjectVal(map[string]*runtime.Value{
                        "role":    runtime.StringVal("user"),
                        "content": runtime.StringVal(prompt),
                }),
        }
        result, err := c.chat(msgs, opts)
        if err != nil {
                return result, err
        }
        if result.Tag == runtime.TypeObject {
                if content, ok := result.ObjVal["content"]; ok {
                        return content, nil
                }
        }
        return result, nil
}

func (c *aiClient) embed(text string) (*runtime.Value, error) {
        payload := map[string]interface{}{
                "input": text,
                "model": "text-embedding-3-small",
        }
        if c.model != "" && strings.Contains(c.model, "embed") {
                payload["model"] = c.model
        }
        body, _ := json.Marshal(payload)
        baseURL := c.baseURL
        if baseURL == "" {
                baseURL = "https://api.openai.com"
        }
        req, err := http.NewRequest("POST", baseURL+"/v1/embeddings", bytes.NewReader(body))
        if err != nil {
                return runtime.ArrayVal(nil), err
        }
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("Authorization", "Bearer "+c.apiKey)
        client := &http.Client{Timeout: c.timeout}
        resp, err := client.Do(req)
        if err != nil {
                return runtime.ArrayVal(nil), err
        }
        defer resp.Body.Close()
        respBody, _ := io.ReadAll(resp.Body)
        var result map[string]interface{}
        if err := json.Unmarshal(respBody, &result); err != nil {
                return runtime.ArrayVal(nil), err
        }
        data, ok := result["data"].([]interface{})
        if !ok || len(data) == 0 {
                return runtime.ArrayVal(nil), nil
        }
        embData, _ := data[0].(map[string]interface{})
        if embData == nil {
                return runtime.ArrayVal(nil), nil
        }
        embedding, _ := embData["embedding"].([]interface{})
        out := make([]*runtime.Value, len(embedding))
        for i, v := range embedding {
                if f, ok := v.(float64); ok {
                        out[i] = runtime.NumberVal(f)
                } else {
                        out[i] = runtime.NumberVal(0)
                }
        }
        return runtime.ArrayVal(out), nil
}

func aiClientVal(c *aiClient) *runtime.Value {
        return runtime.ObjectVal(map[string]*runtime.Value{
                "chat": runtime.FuncVal(&runtime.Function{Name: "chat", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) == 0 {
                                return runtime.Null, fmt.Errorf("messages required")
                        }
                        var msgs []*runtime.Value
                        if args[0].Tag == runtime.TypeArray {
                                msgs = args[0].ArrVal
                        }
                        var opts *runtime.Value
                        if len(args) > 1 {
                                opts = args[1]
                        }
                        return c.chat(msgs, opts)
                }}),
                "complete": runtime.FuncVal(&runtime.Function{Name: "complete", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) == 0 {
                                return runtime.StringVal(""), nil
                        }
                        var opts *runtime.Value
                        if len(args) > 1 {
                                opts = args[1]
                        }
                        return c.complete(args[0].ToString(), opts)
                }}),
                "embed": runtime.FuncVal(&runtime.Function{Name: "embed", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) == 0 {
                                return runtime.ArrayVal(nil), nil
                        }
                        return c.embed(args[0].ToString())
                }}),
                "classify": runtime.FuncVal(&runtime.Function{Name: "classify", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) < 2 {
                                return runtime.Null, nil
                        }
                        text := args[0].ToString()
                        var labels []string
                        if args[1].Tag == runtime.TypeArray {
                                for _, l := range args[1].ArrVal {
                                        labels = append(labels, l.ToString())
                                }
                        }
                        prompt := fmt.Sprintf("Classify the following text into one of these categories: %s\n\nText: %s\n\nRespond with only the category name.", strings.Join(labels, ", "), text)
                        result, err := c.complete(prompt, nil)
                        if err != nil {
                                return runtime.Null, err
                        }
                        return runtime.ObjectVal(map[string]*runtime.Value{
                                "label":  result,
                                "labels": args[1],
                        }), nil
                }}),
                "moderate": runtime.FuncVal(&runtime.Function{Name: "moderate", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.ObjectVal(map[string]*runtime.Value{
                                "flagged":    runtime.False,
                                "categories": runtime.ObjectVal(nil),
                        }), nil
                }}),
                "model":    runtime.StringVal(c.model),
                "provider": runtime.StringVal(c.provider),
        })
}

func AiModule() *runtime.Value {
        defaultClient := &aiClient{
                apiKey:   os.Getenv("OPENAI_API_KEY"),
                baseURL:  os.Getenv("OPENAI_BASE_URL"),
                model:    "gpt-3.5-turbo",
                provider: "openai",
                timeout:  60 * time.Second,
        }
        if replitKey := os.Getenv("REPLIT_AI_KEY"); replitKey != "" {
                defaultClient.apiKey = replitKey
                defaultClient.baseURL = "https://inference.do.repl.it"
                defaultClient.provider = "replit"
        }

        return runtime.ObjectVal(map[string]*runtime.Value{
                "create": runtime.FuncVal(&runtime.Function{Name: "create", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        c := &aiClient{
                                apiKey:   defaultClient.apiKey,
                                baseURL:  defaultClient.baseURL,
                                model:    defaultClient.model,
                                provider: defaultClient.provider,
                                timeout:  defaultClient.timeout,
                        }
                        if len(args) > 0 && args[0].Tag == runtime.TypeObject {
                                opts := args[0].ObjVal
                                if v, ok := opts["apiKey"]; ok {
                                        c.apiKey = v.ToString()
                                }
                                if v, ok := opts["key"]; ok {
                                        c.apiKey = v.ToString()
                                }
                                if v, ok := opts["baseURL"]; ok {
                                        c.baseURL = v.ToString()
                                }
                                if v, ok := opts["model"]; ok {
                                        c.model = v.ToString()
                                }
                                if v, ok := opts["provider"]; ok {
                                        c.provider = v.ToString()
                                }
                                if v, ok := opts["timeout"]; ok {
                                        c.timeout = time.Duration(v.ToNumber()) * time.Millisecond
                                }
                        }
                        return aiClientVal(c), nil
                }}),

                "chat": runtime.FuncVal(&runtime.Function{Name: "chat", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) == 0 {
                                return runtime.Null, fmt.Errorf("messages required")
                        }
                        var msgs []*runtime.Value
                        if args[0].Tag == runtime.TypeArray {
                                msgs = args[0].ArrVal
                        }
                        var opts *runtime.Value
                        if len(args) > 1 {
                                opts = args[1]
                        }
                        return defaultClient.chat(msgs, opts)
                }}),

                "complete": runtime.FuncVal(&runtime.Function{Name: "complete", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) == 0 {
                                return runtime.StringVal(""), nil
                        }
                        var opts *runtime.Value
                        if len(args) > 1 {
                                opts = args[1]
                        }
                        return defaultClient.complete(args[0].ToString(), opts)
                }}),

                "embed": runtime.FuncVal(&runtime.Function{Name: "embed", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) == 0 {
                                return runtime.ArrayVal(nil), nil
                        }
                        return defaultClient.embed(args[0].ToString())
                }}),

                "classify": runtime.FuncVal(&runtime.Function{Name: "classify", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) < 2 {
                                return runtime.Null, nil
                        }
                        fn := aiClientVal(defaultClient).ObjVal["classify"]
                        if fn != nil && fn.FnVal != nil && fn.FnVal.Native != nil {
                                return fn.FnVal.Native(args, nil)
                        }
                        return runtime.Null, nil
                }}),

                "moderate": runtime.FuncVal(&runtime.Function{Name: "moderate", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.ObjectVal(map[string]*runtime.Value{"flagged": runtime.False}), nil
                }}),

                "similarity": runtime.FuncVal(&runtime.Function{Name: "similarity", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) < 2 {
                                return runtime.NumberVal(0), nil
                        }
                        a := args[0]
                        b := args[1]
                        if a.Tag != runtime.TypeArray || b.Tag != runtime.TypeArray {
                                return runtime.NumberVal(0), nil
                        }
                        if len(a.ArrVal) != len(b.ArrVal) {
                                return runtime.NumberVal(0), nil
                        }
                        dot, normA, normB := 0.0, 0.0, 0.0
                        for i := range a.ArrVal {
                                av := a.ArrVal[i].ToNumber()
                                bv := b.ArrVal[i].ToNumber()
                                dot += av * bv
                                normA += av * av
                                normB += bv * bv
                        }
                        if normA == 0 || normB == 0 {
                                return runtime.NumberVal(0), nil
                        }
                        return runtime.NumberVal(dot / (math.Sqrt(normA) * math.Sqrt(normB))), nil
                }}),
        })
}
