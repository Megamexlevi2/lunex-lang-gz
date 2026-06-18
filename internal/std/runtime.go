// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.
//
// std.runtime — runtime introspection and global environment manipulation.
//
// Usage in Lunex:
//   val runtime = @import("std.runtime")
//   val original_log = runtime.getGlobal("io.log")
//   fn print(...args) { original_log(args) }
//   runtime.setGlobal("print", print)
//   runtime.setGlobal("io.log", print)

package std

import (
        "fmt"
        "lunex/internal/meta"
        "lunex/internal/runtime"
)

func RuntimeModule(interp interface {
	SetGlobal(name string, val *runtime.Value)
	GetGlobal(name string) (*runtime.Value, bool)
	GetAllGlobalNames() []string
}) *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		// setGlobal(name, value) — write a value into the global scope.
		// Supports dot-path notation to reach nested objects, e.g. "io.log".
		"setGlobal": runtime.FuncVal(&runtime.Function{
			Name: "setGlobal",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) < 2 {
					return runtime.Undefined, fmt.Errorf("setGlobal: name and value required")
				}
				interp.SetGlobal(args[0].ToString(), args[1])
				return runtime.Undefined, nil
			},
		}),

		// getGlobal(name) — read a value from the global scope.
		// Supports dot-path notation, e.g. "io.log".
		// Returns undefined when the name does not exist.
		"getGlobal": runtime.FuncVal(&runtime.Function{
			Name: "getGlobal",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) == 0 {
					return runtime.Undefined, nil
				}

				val, ok := interp.GetGlobal(args[0].ToString())
				if !ok || val == nil {
					return runtime.Undefined, nil
				}

				return val, nil
			},
		}),

		// hasGlobal(name) — returns true when the name exists in the global scope.
		"hasGlobal": runtime.FuncVal(&runtime.Function{
			Name: "hasGlobal",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) == 0 {
					return runtime.False, nil
				}

				_, ok := interp.GetGlobal(args[0].ToString())
				return runtime.BoolVal(ok), nil
			},
		}),

		// globals() — returns an array of all visible global names.
		"globals": runtime.FuncVal(&runtime.Function{
			Name: "globals",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				names := interp.GetAllGlobalNames()
				out := make([]*runtime.Value, len(names))

				for i, n := range names {
					out[i] = runtime.StringVal(n)
				}

				return runtime.ArrayVal(out), nil
			},
		}),

		// version() — returns the Lunex runtime version sourced from version.json.
		"version": runtime.FuncVal(&runtime.Function{
			Name: "version",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				return runtime.StringVal(meta.Version()), nil
			},
		}),
	})
}