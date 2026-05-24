// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"lunex/internal/runtime"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/cbroglie/mustache"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"
)

func CSVModule() *runtime.Value {
	parse := runtime.FuncVal(&runtime.Function{
		Name: "parse",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), fmt.Errorf("parse(csvString, options?)")
			}
			content := args[0].ToString()
			separator := ','
			hasHeader := true
			if len(args) > 1 && args[1].Tag == runtime.TypeObject {
				opts := args[1].ObjVal
				if v, ok := opts["separator"]; ok && len(v.StrVal) > 0 {
					separator = rune(v.StrVal[0])
				}
				if v, ok := opts["header"]; ok {
					hasHeader = v.BoolVal
				}
			}
			r := csv.NewReader(strings.NewReader(content))
			r.Comma = separator
			r.LazyQuotes = true
			rows, err := r.ReadAll()
			if err != nil {
				return runtime.ArrayVal(nil), err
			}
			if len(rows) == 0 {
				return runtime.ArrayVal(nil), nil
			}
			if hasHeader && len(rows) > 0 {
				headers := rows[0]
				result := make([]*runtime.Value, 0, len(rows)-1)
				for _, row := range rows[1:] {
					obj := make(map[string]*runtime.Value)
					for j, h := range headers {
						val := ""
						if j < len(row) {
							val = row[j]
						}
						obj[h] = runtime.StringVal(val)
					}
					result = append(result, runtime.ObjectVal(obj))
				}
				return runtime.ArrayVal(result), nil
			}
			result := make([]*runtime.Value, len(rows))
			for i, row := range rows {
				cells := make([]*runtime.Value, len(row))
				for j, c := range row {
					cells[j] = runtime.StringVal(c)
				}
				result[i] = runtime.ArrayVal(cells)
			}
			return runtime.ArrayVal(result), nil
		},
	})

	stringify := runtime.FuncVal(&runtime.Function{
		Name: "stringify",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeArray {
				return runtime.StringVal(""), fmt.Errorf("stringify(array, options?)")
			}
			separator := ','
			if len(args) > 1 && args[1].Tag == runtime.TypeObject {
				if v, ok := args[1].ObjVal["separator"]; ok && len(v.StrVal) > 0 {
					separator = rune(v.StrVal[0])
				}
			}
			var buf bytes.Buffer
			w := csv.NewWriter(&buf)
			w.Comma = separator
			// Collect column order from the first object row (stable, sorted)
			var objKeys []string
			for _, rowVal := range args[0].ArrVal {
				if rowVal.Tag == runtime.TypeObject && objKeys == nil {
					objKeys = make([]string, 0, len(rowVal.ObjVal))
					for k := range rowVal.ObjVal {
						objKeys = append(objKeys, k)
					}
					sort.Strings(objKeys)
				}
			}
			for _, rowVal := range args[0].ArrVal {
				if rowVal.Tag == runtime.TypeArray {
					row := make([]string, len(rowVal.ArrVal))
					for i, cell := range rowVal.ArrVal {
						row[i] = cell.ToString()
					}
					w.Write(row)
				} else if rowVal.Tag == runtime.TypeObject {
					keys := objKeys
					if keys == nil {
						keys = make([]string, 0, len(rowVal.ObjVal))
						for k := range rowVal.ObjVal {
							keys = append(keys, k)
						}
						sort.Strings(keys)
					}
					row := make([]string, len(keys))
					for i, k := range keys {
						if v, ok := rowVal.ObjVal[k]; ok {
							row[i] = v.ToString()
						}
					}
					w.Write(row)
				}
			}
			w.Flush()
			return runtime.StringVal(buf.String()), w.Error()
		},
	})

	readFile := runtime.FuncVal(&runtime.Function{
		Name: "readFile",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), fmt.Errorf("readFile(path, options?)")
			}
			data, err := os.ReadFile(args[0].ToString())
			if err != nil {
				return runtime.ArrayVal(nil), err
			}
			var parseArgs []*runtime.Value
			parseArgs = append(parseArgs, runtime.StringVal(string(data)))
			if len(args) > 1 {
				parseArgs = append(parseArgs, args[1])
			}
			return CSVModule().ObjVal["parse"].FnVal.Native(parseArgs, nil)
		},
	})

	writeFile := runtime.FuncVal(&runtime.Function{
		Name: "writeFile",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("writeFile(path, data, options?)")
			}
			var stringifyArgs []*runtime.Value
			stringifyArgs = append(stringifyArgs, args[1])
			if len(args) > 2 {
				stringifyArgs = append(stringifyArgs, args[2])
			}
			result, err := CSVModule().ObjVal["stringify"].FnVal.Native(stringifyArgs, nil)
			if err != nil {
				return runtime.Null, err
			}
			err = os.WriteFile(args[0].ToString(), []byte(result.StrVal), 0644)
			return runtime.BoolVal(err == nil), err
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"parse":     parse,
		"stringify": stringify,
		"readFile":  readFile,
		"writeFile": writeFile,
	})
}

func YAMLModule() *runtime.Value {
	parse := runtime.FuncVal(&runtime.Function{
		Name: "parse",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("parse(yamlString)")
			}
			var raw interface{}
			if err := yaml.Unmarshal([]byte(args[0].ToString()), &raw); err != nil {
				return runtime.Null, err
			}
			return jsonToValue(raw), nil
		},
	})

	stringify := runtime.FuncVal(&runtime.Function{
		Name: "stringify",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			native := ntlToGoNative(args[0])
			out, err := yaml.Marshal(native)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(string(out)), nil
		},
	})

	readFile := runtime.FuncVal(&runtime.Function{
		Name: "readFile",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("readFile(path)")
			}
			data, err := os.ReadFile(args[0].ToString())
			if err != nil {
				return runtime.Null, err
			}
			var raw interface{}
			if err := yaml.Unmarshal(data, &raw); err != nil {
				return runtime.Null, err
			}
			return jsonToValue(raw), nil
		},
	})

	writeFile := runtime.FuncVal(&runtime.Function{
		Name: "writeFile",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("writeFile(path, data)")
			}
			native := ntlToGoNative(args[1])
			out, err := yaml.Marshal(native)
			if err != nil {
				return runtime.Null, err
			}
			err = os.WriteFile(args[0].ToString(), out, 0644)
			return runtime.BoolVal(err == nil), err
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"parse":     parse,
		"stringify": stringify,
		"readFile":  readFile,
		"writeFile": writeFile,
	})
}

func TOMLModule() *runtime.Value {
	parse := runtime.FuncVal(&runtime.Function{
		Name: "parse",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("parse(tomlString)")
			}
			var raw interface{}
			if _, err := toml.Decode(args[0].ToString(), &raw); err != nil {
				return runtime.Null, err
			}
			return jsonToValue(raw), nil
		},
	})

	stringify := runtime.FuncVal(&runtime.Function{
		Name: "stringify",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			native := ntlToGoNative(args[0])
			var buf bytes.Buffer
			if err := toml.NewEncoder(&buf).Encode(native); err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(buf.String()), nil
		},
	})

	readFile := runtime.FuncVal(&runtime.Function{
		Name: "readFile",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("readFile(path)")
			}
			var raw interface{}
			if _, err := toml.DecodeFile(args[0].ToString(), &raw); err != nil {
				return runtime.Null, err
			}
			return jsonToValue(raw), nil
		},
	})

	writeFile := runtime.FuncVal(&runtime.Function{
		Name: "writeFile",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("writeFile(path, data)")
			}
			native := ntlToGoNative(args[1])
			var buf bytes.Buffer
			if err := toml.NewEncoder(&buf).Encode(native); err != nil {
				return runtime.Null, err
			}
			err := os.WriteFile(args[0].ToString(), buf.Bytes(), 0644)
			return runtime.BoolVal(err == nil), err
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"parse":     parse,
		"stringify": stringify,
		"readFile":  readFile,
		"writeFile": writeFile,
	})
}

func MarkdownModule() *runtime.Value {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Table, extension.Strikethrough),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithUnsafe()),
	)

	toHTML := runtime.FuncVal(&runtime.Function{
		Name: "toHTML",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			var buf bytes.Buffer
			if err := md.Convert([]byte(args[0].ToString()), &buf); err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(buf.String()), nil
		},
	})

	readFile := runtime.FuncVal(&runtime.Function{
		Name: "readFile",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("readFile(path)")
			}
			data, err := os.ReadFile(args[0].ToString())
			if err != nil {
				return runtime.Null, err
			}
			var buf bytes.Buffer
			if err := md.Convert(data, &buf); err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(buf.String()), nil
		},
	})

	renderFile := runtime.FuncVal(&runtime.Function{
		Name: "renderFile",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("renderFile(inputPath, outputPath)")
			}
			data, err := os.ReadFile(args[0].ToString())
			if err != nil {
				return runtime.Null, err
			}
			var buf bytes.Buffer
			if err := md.Convert(data, &buf); err != nil {
				return runtime.Null, err
			}
			err = os.WriteFile(args[1].ToString(), buf.Bytes(), 0644)
			return runtime.BoolVal(err == nil), err
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"toHTML":     toHTML,
		"parse":      toHTML,
		"readFile":   readFile,
		"renderFile": renderFile,
	})
}

func MustacheModule() *runtime.Value {
	render := runtime.FuncVal(&runtime.Function{
		Name: "render",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.StringVal(""), fmt.Errorf("render(template, data)")
			}
			tmpl := args[0].ToString()
			data := ntlToGoNative(args[1])
			result, err := mustache.Render(tmpl, data)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(result), nil
		},
	})

	renderFile := runtime.FuncVal(&runtime.Function{
		Name: "renderFile",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("renderFile(path, data)")
			}
			data := ntlToGoNative(args[1])
			result, err := mustache.RenderFile(args[0].ToString(), data)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(result), nil
		},
	})

	parse := runtime.FuncVal(&runtime.Function{
		Name: "parse",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("parse(template)")
			}
			tmpl, err := mustache.ParseString(args[0].ToString())
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"render": runtime.FuncVal(&runtime.Function{
					Name: "render",
					Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
						if len(a) == 0 {
							return runtime.StringVal(""), nil
						}
						result, err := tmpl.Render(ntlToGoNative(a[0]))
						if err != nil {
							return runtime.Null, err
						}
						return runtime.StringVal(result), nil
					},
				}),
			}), nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"render":     render,
		"renderFile": renderFile,
		"parse":      parse,
	})
}

func ntlToGoNative(v *runtime.Value) interface{} {
	if v == nil {
		return nil
	}
	switch v.Tag {
	case runtime.TypeNull, runtime.TypeUndefined:
		return nil
	case runtime.TypeBool:
		return v.BoolVal
	case runtime.TypeNumber:
		return v.NumVal
	case runtime.TypeString:
		return v.StrVal
	case runtime.TypeArray:
		arr := make([]interface{}, len(v.ArrVal))
		for i, el := range v.ArrVal {
			arr[i] = ntlToGoNative(el)
		}
		return arr
	case runtime.TypeObject:
		obj := make(map[string]interface{}, len(v.ObjVal))
		for k, el := range v.ObjVal {
			if el != nil && el.Tag != runtime.TypeFunction {
				obj[k] = ntlToGoNative(el)
			}
		}
		return obj
	default:
		return v.ToString()
	}
}
