// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
        "encoding/json"
        "fmt"
        "lunex/internal/runtime"
        "strings"
)

func NativeModule() *runtime.Value {
        return runtime.ObjectVal(map[string]*runtime.Value{
                "isString": runtime.FuncVal(&runtime.Function{Name: "isString", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.BoolVal(len(args) > 0 && args[0].Tag == runtime.TypeString), nil
                }}),
                "isNumber": runtime.FuncVal(&runtime.Function{Name: "isNumber", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.BoolVal(len(args) > 0 && args[0].Tag == runtime.TypeNumber), nil
                }}),
                "isBool": runtime.FuncVal(&runtime.Function{Name: "isBool", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.BoolVal(len(args) > 0 && args[0].Tag == runtime.TypeBool), nil
                }}),
                "isArray": runtime.FuncVal(&runtime.Function{Name: "isArray", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.BoolVal(len(args) > 0 && args[0].Tag == runtime.TypeArray), nil
                }}),
                "isObject": runtime.FuncVal(&runtime.Function{Name: "isObject", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.BoolVal(len(args) > 0 && args[0].Tag == runtime.TypeObject), nil
                }}),
                "isNull": runtime.FuncVal(&runtime.Function{Name: "isNull", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.BoolVal(len(args) > 0 && args[0].Tag == runtime.TypeNull), nil
                }}),
                "isUndefined": runtime.FuncVal(&runtime.Function{Name: "isUndefined", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.BoolVal(len(args) == 0 || args[0].Tag == runtime.TypeUndefined), nil
                }}),
                "isFunction": runtime.FuncVal(&runtime.Function{Name: "isFunction", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.BoolVal(len(args) > 0 && args[0].Tag == runtime.TypeFunction), nil
                }}),
                "typeOf": runtime.FuncVal(&runtime.Function{
                        Name: "typeOf",
                        Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                                if len(args) == 0 {
                                        return runtime.StringVal("undefined"), nil
                                }
                                return runtime.StringVal(getTypeName(args[0])), nil
                        },
                }),
                "toJSON": runtime.FuncVal(&runtime.Function{Name: "toJSON", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) == 0 {
                                return runtime.StringVal("null"), nil
                        }
                        return runtime.StringVal(valueToJSON(args[0])), nil
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
                "deepClone": runtime.FuncVal(&runtime.Function{Name: "deepClone", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) == 0 {
                                return runtime.Undefined, nil
                        }
                        return deepCopy(args[0]), nil
                }}),
                "deepEqual": runtime.FuncVal(&runtime.Function{Name: "deepEqual", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) < 2 {
                                return runtime.False, nil
                        }
                        return runtime.BoolVal(deepEqual(args[0], args[1])), nil
                }}),
        })
}

func getTypeName(v *runtime.Value) string {
        if v == nil {
                return "undefined"
        }
        switch v.Tag {
        case runtime.TypeString:
                return "string"
        case runtime.TypeNumber:
                return "number"
        case runtime.TypeBool:
                return "boolean"
        case runtime.TypeNull:
                return "null"
        case runtime.TypeArray:
                return "array"
        case runtime.TypeFunction:
                return "function"
        case runtime.TypeObject:
                return "object"
        case runtime.TypeClass:
                return "class"
        case runtime.TypeInstance:
                return "object"
        default:
                return "undefined"
        }
}

func sprintArgs(args []*runtime.Value) string {
        parts := make([]string, 0, len(args))
        for _, a := range args {
                if a == nil {
                        parts = append(parts, "undefined")
                        continue
                }
                if a.Tag == runtime.TypeString {
                        parts = append(parts, a.StrVal)
                } else {
                        parts = append(parts, a.Inspect())
                }
        }
        return strings.Join(parts, " ")
}

func valueToJSON(val *runtime.Value) string {
        if val == nil {
                return "null"
        }
        switch val.Tag {
        case runtime.TypeNull, runtime.TypeUndefined:
                return "null"
        case runtime.TypeBool:
                if val.BoolVal {
                        return "true"
                }
                return "false"
        case runtime.TypeNumber:
                return val.ToString()
        case runtime.TypeString:
                return fmt.Sprintf("%q", val.StrVal)
        case runtime.TypeArray:
                parts := make([]string, 0, len(val.ArrVal))
                for _, el := range val.ArrVal {
                        parts = append(parts, valueToJSON(el))
                }
                return "[" + strings.Join(parts, ",") + "]"
        case runtime.TypeObject:
                parts := make([]string, 0, len(val.ObjVal))
                keys := make([]string, 0, len(val.ObjVal))
                for k := range val.ObjVal {
                        keys = append(keys, k)
                }
                for _, k := range keys {
                        v := val.ObjVal[k]
                        if v != nil && v.Tag != runtime.TypeFunction {
                                parts = append(parts, fmt.Sprintf("%q:%s", k, valueToJSON(v)))
                        }
                }
                return "{" + strings.Join(parts, ",") + "}"
        default:
                return "null"
        }
}

func parseJSON(s string) (*runtime.Value, error) {
        s = strings.TrimSpace(s)
        if s == "" {
                return runtime.Null, nil
        }
        var raw interface{}
        if err := json.Unmarshal([]byte(s), &raw); err != nil {
                return runtime.StringVal(s), nil
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
                return runtime.StringVal(fmt.Sprintf("%v", v))
        }
}

func deepCopy(v *runtime.Value) *runtime.Value {
        if v == nil {
                return runtime.Undefined
        }
        switch v.Tag {
        case runtime.TypeArray:
                out := make([]*runtime.Value, len(v.ArrVal))
                for i, el := range v.ArrVal {
                        out[i] = deepCopy(el)
                }
                return runtime.ArrayVal(out)
        case runtime.TypeObject:
                out := make(map[string]*runtime.Value, len(v.ObjVal))
                for k, el := range v.ObjVal {
                        out[k] = deepCopy(el)
                }
                return runtime.ObjectVal(out)
        default:
                return v
        }
}

func deepEqual(a, b *runtime.Value) bool {
        if a == nil && b == nil {
                return true
        }
        if a == nil || b == nil {
                return false
        }
        if a.Tag != b.Tag {
                return false
        }
        switch a.Tag {
        case runtime.TypeArray:
                if len(a.ArrVal) != len(b.ArrVal) {
                        return false
                }
                for i := range a.ArrVal {
                        if !deepEqual(a.ArrVal[i], b.ArrVal[i]) {
                                return false
                        }
                }
                return true
        case runtime.TypeObject:
                if len(a.ObjVal) != len(b.ObjVal) {
                        return false
                }
                for k, va := range a.ObjVal {
                        if !deepEqual(va, b.ObjVal[k]) {
                                return false
                        }
                }
                return true
        default:
                return a.StrictEquals(b)
        }
}

func flattenValue(v *runtime.Value, depth int) *runtime.Value {
        if v == nil || v.Tag != runtime.TypeArray {
                return runtime.ArrayVal(nil)
        }
        var out []*runtime.Value
        for _, el := range v.ArrVal {
                if el != nil && el.Tag == runtime.TypeArray && depth > 0 {
                        inner := flattenValue(el, depth-1)
                        out = append(out, inner.ArrVal...)
                } else {
                        out = append(out, el)
                }
        }
        return runtime.ArrayVal(out)
}
