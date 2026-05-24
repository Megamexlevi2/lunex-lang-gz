package builtin

import (
	"lunex/internal/runtime"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
)

func PathModule() *runtime.Value {
	sep := string(os.PathSeparator)

	return runtime.ObjectVal(map[string]*runtime.Value{
		"sep":       runtime.StringVal(sep),
		"delimiter": runtime.StringVal(string(os.PathListSeparator)),

		"join": runtime.FuncVal(&runtime.Function{Name: "join", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			parts := make([]string, 0, len(args))
			for _, a := range args {
				if a.Tag == runtime.TypeArray {
					for _, el := range a.ArrVal {
						if el != nil {
							parts = append(parts, el.ToString())
						}
					}
				} else {
					parts = append(parts, a.ToString())
				}
			}
			return runtime.StringVal(filepath.Join(parts...)), nil
		}}),

		"resolve": runtime.FuncVal(&runtime.Function{Name: "resolve", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			parts := make([]string, 0, len(args)+1)
			cwd, _ := os.Getwd()
			parts = append(parts, cwd)
			for _, a := range args {
				parts = append(parts, a.ToString())
			}
			result := filepath.Join(parts...)
			abs, err := filepath.Abs(result)
			if err != nil {
				return runtime.StringVal(result), nil
			}
			return runtime.StringVal(abs), nil
		}}),

		"dirname": runtime.FuncVal(&runtime.Function{Name: "dirname", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("."), nil
			}
			return runtime.StringVal(filepath.Dir(args[0].ToString())), nil
		}}),

		"basename": runtime.FuncVal(&runtime.Function{Name: "basename", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			base := filepath.Base(args[0].ToString())
			if len(args) > 1 {
				ext := args[1].ToString()
				base = strings.TrimSuffix(base, ext)
			}
			return runtime.StringVal(base), nil
		}}),

		"extname": runtime.FuncVal(&runtime.Function{Name: "extname", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(filepath.Ext(args[0].ToString())), nil
		}}),

		"isAbsolute": runtime.FuncVal(&runtime.Function{Name: "isAbsolute", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			return runtime.BoolVal(filepath.IsAbs(args[0].ToString())), nil
		}}),

		"normalize": runtime.FuncVal(&runtime.Function{Name: "normalize", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("."), nil
			}
			return runtime.StringVal(filepath.Clean(args[0].ToString())), nil
		}}),

		"relative": runtime.FuncVal(&runtime.Function{Name: "relative", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.StringVal("."), nil
			}
			rel, err := filepath.Rel(args[0].ToString(), args[1].ToString())
			if err != nil {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(rel), nil
		}}),

		"parse": runtime.FuncVal(&runtime.Function{Name: "parse", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ObjectVal(nil), nil
			}
			p := args[0].ToString()
			dir := filepath.Dir(p)
			base := filepath.Base(p)
			ext := filepath.Ext(p)
			name := strings.TrimSuffix(base, ext)
			return runtime.ObjectVal(map[string]*runtime.Value{
				"root": runtime.StringVal(getRootComponent(p)),
				"dir":  runtime.StringVal(dir),
				"base": runtime.StringVal(base),
				"ext":  runtime.StringVal(ext),
				"name": runtime.StringVal(name),
			}), nil
		}}),

		"format": runtime.FuncVal(&runtime.Function{Name: "format", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.StringVal(""), nil
			}
			obj := args[0].ObjVal
			dir := ""
			base := ""
			ext := ""
			name := ""
			if v, ok := obj["dir"]; ok {
				dir = v.ToString()
			}
			if v, ok := obj["base"]; ok {
				base = v.ToString()
			}
			if v, ok := obj["ext"]; ok {
				ext = v.ToString()
			}
			if v, ok := obj["name"]; ok {
				name = v.ToString()
			}
			if base == "" {
				base = name + ext
			}
			if dir != "" && base != "" {
				return runtime.StringVal(filepath.Join(dir, base)), nil
			}
			return runtime.StringVal(base), nil
		}}),

		"toURL": runtime.FuncVal(&runtime.Function{Name: "toURL", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			p := filepath.ToSlash(args[0].ToString())
			if !strings.HasPrefix(p, "/") {
				p = "/" + p
			}
			return runtime.StringVal("file://" + p), nil
		}}),

		"fromURL": runtime.FuncVal(&runtime.Function{Name: "fromURL", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			u := args[0].ToString()
			u = strings.TrimPrefix(u, "file://")
			return runtime.StringVal(filepath.FromSlash(u)), nil
		}}),

		"split": runtime.FuncVal(&runtime.Function{Name: "split", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), nil
			}
			parts := strings.Split(filepath.ToSlash(args[0].ToString()), "/")
			out := make([]*runtime.Value, 0, len(parts))
			for _, part := range parts {
				if part != "" {
					out = append(out, runtime.StringVal(part))
				}
			}
			return runtime.ArrayVal(out), nil
		}}),

		"windows": runtime.FuncVal(&runtime.Function{Name: "windows", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(filepath.FromSlash(args[0].ToString())), nil
		}}),

		"posix": runtime.FuncVal(&runtime.Function{Name: "posix", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(filepath.ToSlash(args[0].ToString())), nil
		}}),

		"cwd": runtime.FuncVal(&runtime.Function{Name: "cwd", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			cwd, _ := os.Getwd()
			return runtime.StringVal(cwd), nil
		}}),

		"home": runtime.FuncVal(&runtime.Function{Name: "home", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			home, _ := os.UserHomeDir()
			return runtime.StringVal(home), nil
		}}),

		"temp": runtime.FuncVal(&runtime.Function{Name: "temp", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.StringVal(os.TempDir()), nil
		}}),

		"exists": runtime.FuncVal(&runtime.Function{Name: "exists", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			_, err := os.Stat(args[0].ToString())
			return runtime.BoolVal(err == nil), nil
		}}),

		"isFile": runtime.FuncVal(&runtime.Function{Name: "isFile", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			info, err := os.Stat(args[0].ToString())
			if err != nil {
				return runtime.False, nil
			}
			return runtime.BoolVal(!info.IsDir()), nil
		}}),

		"isDir": runtime.FuncVal(&runtime.Function{Name: "isDir", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			info, err := os.Stat(args[0].ToString())
			if err != nil {
				return runtime.False, nil
			}
			return runtime.BoolVal(info.IsDir()), nil
		}}),

		"platform": runtime.StringVal(goruntime.GOOS),
		"arch":     runtime.StringVal(goruntime.GOARCH),
	})
}

func getRootComponent(p string) string {
	vol := filepath.VolumeName(p)
	if vol != "" {
		return vol + string(os.PathSeparator)
	}
	if strings.HasPrefix(p, "/") {
		return "/"
	}
	return ""
}
