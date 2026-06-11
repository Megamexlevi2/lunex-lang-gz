// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package std

import (
	"lunex/internal/runtime"
	"os"
	"strings"
)

func EnvModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"get": runtime.FuncVal(&runtime.Function{Name: "get", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Undefined, nil
			}
			val, ok := os.LookupEnv(args[0].ToString())
			if !ok {
				if len(args) > 1 {
					return args[1], nil
				}
				return runtime.Undefined, nil
			}
			return runtime.StringVal(val), nil
		}}),

		"set": runtime.FuncVal(&runtime.Function{Name: "set", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) >= 2 {
				os.Setenv(args[0].ToString(), args[1].ToString())
			}
			return runtime.Undefined, nil
		}}),

		"has": runtime.FuncVal(&runtime.Function{Name: "has", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			_, ok := os.LookupEnv(args[0].ToString())
			return runtime.BoolVal(ok), nil
		}}),

		"delete": runtime.FuncVal(&runtime.Function{Name: "delete", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				os.Unsetenv(args[0].ToString())
			}
			return runtime.Undefined, nil
		}}),

		"all": runtime.FuncVal(&runtime.Function{Name: "all", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			env := os.Environ()
			out := make(map[string]*runtime.Value, len(env))
			for _, e := range env {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 {
					out[parts[0]] = runtime.StringVal(parts[1])
				}
			}
			return runtime.ObjectVal(out), nil
		}}),

		"load": runtime.FuncVal(&runtime.Function{Name: "load", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			path := ".env"
			if len(args) > 0 {
				path = args[0].ToString()
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return runtime.False, nil
			}
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
						val = val[1 : len(val)-1]
					}
					os.Setenv(key, val)
				}
			}
			return runtime.True, nil
		}}),

		"require": runtime.FuncVal(&runtime.Function{Name: "require", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Undefined, nil
			}
			key := args[0].ToString()
			val, ok := os.LookupEnv(key)
			if !ok || val == "" {
				return runtime.Undefined, nil
			}
			return runtime.StringVal(val), nil
		}}),

		"int": runtime.FuncVal(&runtime.Function{Name: "int", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(0), nil
			}
			val, ok := os.LookupEnv(args[0].ToString())
			if !ok {
				if len(args) > 1 {
					return args[1], nil
				}
				return runtime.NumberVal(0), nil
			}
			v := runtime.StringVal(val)
			return runtime.NumberVal(v.ToNumber()), nil
		}}),

		"bool": runtime.FuncVal(&runtime.Function{Name: "bool", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			val, ok := os.LookupEnv(args[0].ToString())
			if !ok {
				if len(args) > 1 {
					return args[1], nil
				}
				return runtime.False, nil
			}
			lower := strings.ToLower(strings.TrimSpace(val))
			return runtime.BoolVal(lower == "true" || lower == "1" || lower == "yes" || lower == "on"), nil
		}}),
	})
}
