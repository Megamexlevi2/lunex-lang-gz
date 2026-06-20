// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package std

import (
	"encoding/json"
	"lunex/internal/runtime"
	"math"
	"os"
	"sort"
	"strings"
)

func JsonModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"parse": runtime.FuncVal(&runtime.Function{Name: "parse", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, nil
			}
			v, err := parseJSON(args[0].ToString())
			if err != nil {
				return runtime.Null, nil
			}
			return v, nil
		}}),

		"stringify": runtime.FuncVal(&runtime.Function{Name: "stringify", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("null"), nil
			}
			indent := 2
			if len(args) > 1 {
				n := int(args[1].ToNumber())
				if n >= 0 {
					indent = n
				}
			}
			out, err := stringifyJSON(args[0], indent)
			if err != nil {
				return runtime.Null, nil
			}
			return runtime.StringVal(out), nil
		}}),

		"pretty": runtime.FuncVal(&runtime.Function{Name: "pretty", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("null"), nil
			}
			indent := 2
			if len(args) > 1 {
				n := int(args[1].ToNumber())
				if n >= 0 {
					indent = n
				}
			}
			out, err := stringifyJSON(args[0], indent)
			if err != nil {
				return runtime.Null, nil
			}
			return runtime.StringVal(out), nil
		}}),

		"compact": runtime.FuncVal(&runtime.Function{Name: "compact", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("null"), nil
			}
			return runtime.StringVal(valueToJSON(args[0])), nil
		}}),

		"isValid": runtime.FuncVal(&runtime.Function{Name: "isValid", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			return runtime.BoolVal(json.Valid([]byte(args[0].ToString()))), nil
		}}),

		"toJSON": runtime.FuncVal(&runtime.Function{Name: "toJSON", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal("null"), nil
			}
			out, err := stringifyJSON(args[0], 2)
			if err != nil {
				return runtime.Null, nil
			}
			return runtime.StringVal(out), nil
		}}),

		"fromJSON": runtime.FuncVal(&runtime.Function{Name: "fromJSON", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, nil
			}
			v, err := parseJSON(args[0].ToString())
			if err != nil {
				return runtime.Null, nil
			}
			return v, nil
		}}),

		"save": runtime.FuncVal(&runtime.Function{Name: "save", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, nil
			}
			indent := 2
			if len(args) > 2 {
				n := int(args[2].ToNumber())
				if n >= 0 {
					indent = n
				}
			}
			out, err := stringifyJSON(args[1], indent)
			if err != nil {
				return runtime.False, nil
			}
			err = os.WriteFile(args[0].ToString(), []byte(out), 0644)
			return runtime.BoolVal(err == nil), nil
		}}),

		"writeFile": runtime.FuncVal(&runtime.Function{Name: "writeFile", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, nil
			}
			indent := 2
			if len(args) > 2 {
				n := int(args[2].ToNumber())
				if n >= 0 {
					indent = n
				}
			}
			out, err := stringifyJSON(args[1], indent)
			if err != nil {
				return runtime.False, nil
			}
			err = os.WriteFile(args[0].ToString(), []byte(out), 0644)
			return runtime.BoolVal(err == nil), nil
		}}),
	})
}

func stringifyJSON(v *runtime.Value, indentSpaces int) (string, error) {
	if v == nil {
		return "null", nil
	}
	if indentSpaces < 0 {
		indentSpaces = 0
	}
	data, include := jsonValueData(v, false)
	if !include {
		return "null", nil
	}
	if indentSpaces == 0 {
		out, err := json.Marshal(data)
		if err != nil {
			return "null", err
		}
		return string(out), nil
	}
	out, err := json.MarshalIndent(data, "", strings.Repeat(" ", indentSpaces))
	if err != nil {
		return "null", err
	}
	return string(out), nil
}

func jsonValueData(v *runtime.Value, inArray bool) (interface{}, bool) {
	if v == nil {
		if inArray {
			return nil, true
		}
		return nil, false
	}
	switch v.Tag {
	case runtime.TypeNull, runtime.TypeUndefined:
		if inArray {
			return nil, true
		}
		return nil, false
	case runtime.TypeBool:
		return v.BoolVal, true
	case runtime.TypeNumber:
		if math.IsNaN(v.NumVal) || math.IsInf(v.NumVal, 0) {
			if inArray {
				return nil, true
			}
			return nil, false
		}
		return v.NumVal, true
	case runtime.TypeString:
		return v.StrVal, true
	case runtime.TypeArray:
		arr := make([]interface{}, len(v.ArrVal))
		for i, el := range v.ArrVal {
			child, ok := jsonValueData(el, true)
			if !ok {
				child = nil
			}
			arr[i] = child
		}
		return arr, true
	case runtime.TypeObject:
		keys := make([]string, 0, len(v.ObjVal))
		for k := range v.ObjVal {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		obj := make(map[string]interface{}, len(v.ObjVal))
		for _, k := range keys {
			child, ok := jsonValueData(v.ObjVal[k], false)
			if !ok {
				continue
			}
			obj[k] = child
		}
		return obj, true
	default:
		if inArray {
			return nil, true
		}
		return nil, false
	}
}

func valueToJSON(v *runtime.Value) string {
	out, err := stringifyJSON(v, 0)
	if err != nil {
		return "null"
	}
	return out
}

func parseJSON(s string) (*runtime.Value, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return runtime.Null, nil
	}
	var raw interface{}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return runtime.Null, err
	}
	return jsonToValue(raw), nil
}

func jsonToValue(v interface{}) *runtime.Value {
	if v == nil {
		return runtime.Null
	}
	switch val := v.(type) {
	case bool:
		return runtime.BoolVal(val)
	case float64:
		return runtime.NumberVal(val)
	case string:
		return runtime.StringVal(val)
	case []interface{}:
		arr := make([]*runtime.Value, len(val))
		for i, el := range val {
			arr[i] = jsonToValue(el)
		}
		return runtime.ArrayVal(arr)
	case map[string]interface{}:
		obj := make(map[string]*runtime.Value, len(val))
		for k, el := range val {
			obj[k] = jsonToValue(el)
		}
		return runtime.ObjectVal(obj)
	default:
		return runtime.Null
	}
}
