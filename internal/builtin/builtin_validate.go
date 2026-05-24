// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

  import (
  	"fmt"
  	"lunex/internal/runtime"
  	"regexp"
  	"strings"
  )

  var (
  	emailRegex     = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$`)
  	urlRegex       = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
  	ipv4Regex      = regexp.MustCompile(`^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$`)
  	ipv6Regex      = regexp.MustCompile(`^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`)
  	hexRegex       = regexp.MustCompile(`^[0-9a-fA-F]+$`)
  	alphaRegex     = regexp.MustCompile(`^[a-zA-Z]+$`)
  	alphaNumRegex  = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
  	uuidRegex      = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
  	creditCardRegex = regexp.MustCompile(`^\d{13,19}$`)
  	phoneRegex     = regexp.MustCompile(`^\+?[\d\s\-\(\)]{7,20}$`)
  	slugRegex      = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
  	base64Regex    = regexp.MustCompile(`^[A-Za-z0-9+/]+=*$`)
  )

  func validateSchema(val *runtime.Value, schema *runtime.Value) (bool, string) {
  	if schema == nil || schema.Tag != runtime.TypeObject {
  		return true, ""
  	}
  	schemaType := ""
  	if t, ok := schema.ObjVal["type"]; ok && t != nil {
  		schemaType = t.ToString()
  	}
  	if schemaType != "" && schemaType != "any" {
  		actualType := getTypeName(val)
  		if schemaType == "date" {
  			if actualType != "number" && actualType != "string" {
  				return false, "expected date (number or string), got " + actualType
  			}
  		} else if actualType != schemaType {
  			return false, "expected " + schemaType + ", got " + actualType
  		}
  	}
  	if req, ok := schema.ObjVal["required"]; ok && req != nil && req.BoolVal {
  		if val == nil || val.IsNullish() {
  			return false, "value is required"
  		}
  	}
  	if val == nil || val.IsNullish() {
  		return true, ""
  	}
  	if schemaType == "number" || val.Tag == runtime.TypeNumber {
  		n := val.ToNumber()
  		if mn, ok := schema.ObjVal["min"]; ok && mn != nil {
  			if n < mn.ToNumber() {
  				return false, "value must be >= " + mn.ToString()
  			}
  		}
  		if mx, ok := schema.ObjVal["max"]; ok && mx != nil {
  			if n > mx.ToNumber() {
  				return false, "value must be <= " + mx.ToString()
  			}
  		}
  	}
  	if schemaType == "string" || val.Tag == runtime.TypeString {
  		s := val.ToString()
  		if mn, ok := schema.ObjVal["minLength"]; ok && mn != nil {
  			if len(s) < int(mn.ToNumber()) {
  				return false, "value must be at least " + mn.ToString() + " characters"
  			}
  		}
  		if mx, ok := schema.ObjVal["maxLength"]; ok && mx != nil {
  			if len(s) > int(mx.ToNumber()) {
  				return false, "value must be at most " + mx.ToString() + " characters"
  			}
  		}
  		if pat, ok := schema.ObjVal["pattern"]; ok && pat != nil {
  			re, err := regexp.Compile(pat.ToString())
  			if err == nil && !re.MatchString(s) {
  				return false, "value does not match pattern"
  			}
  		}
  		if fmtVal, ok := schema.ObjVal["format"]; ok && fmtVal != nil {
  			switch fmtVal.ToString() {
  			case "email":
  				if !emailRegex.MatchString(s) {
  					return false, "value must be a valid email"
  				}
  			case "url":
  				if !urlRegex.MatchString(s) {
  					return false, "value must be a valid URL"
  				}
  			case "uuid":
  				if !uuidRegex.MatchString(strings.ToLower(s)) {
  					return false, "value must be a valid UUID"
  				}
  			case "ipv4":
  				if !ipv4Regex.MatchString(s) {
  					return false, "value must be a valid IPv4 address"
  				}
  			}
  		}
  	}
  	if enum, ok := schema.ObjVal["enum"]; ok && enum != nil && enum.Tag == runtime.TypeArray {
  		found := false
  		for _, item := range enum.ArrVal {
  			if val.StrictEquals(item) {
  				found = true
  				break
  			}
  		}
  		if !found {
  			opts := make([]string, len(enum.ArrVal))
  			for i, item := range enum.ArrVal {
  				opts[i] = item.ToString()
  			}
  			return false, "value must be one of: " + strings.Join(opts, ", ")
  		}
  	}
  	if schemaType == "object" || val.Tag == runtime.TypeObject {
  		if props, ok := schema.ObjVal["properties"]; ok && props != nil && props.Tag == runtime.TypeObject {
  			for field, fieldSchema := range props.ObjVal {
  				fieldVal := val.ObjVal[field]
  				if fieldVal == nil {
  					fieldVal = runtime.Undefined
  				}
  				if ok, msg := validateSchema(fieldVal, fieldSchema); !ok {
  					return false, "field '" + field + "': " + msg
  				}
  			}
  		}
  	}
  	if schemaType == "array" || val.Tag == runtime.TypeArray {
  		if items, ok := schema.ObjVal["items"]; ok && items != nil {
  			for i, item := range val.ArrVal {
  				if ok, msg := validateSchema(item, items); !ok {
  					return false, fmt.Sprintf("item[%d]: %s", i, msg)
  				}
  			}
  		}
  		if mn, ok := schema.ObjVal["minItems"]; ok && mn != nil {
  			if len(val.ArrVal) < int(mn.ToNumber()) {
  				return false, "array must have at least " + mn.ToString() + " items"
  			}
  		}
  		if mx, ok := schema.ObjVal["maxItems"]; ok && mx != nil {
  			if len(val.ArrVal) > int(mx.ToNumber()) {
  				return false, "array must have at most " + mx.ToString() + " items"
  			}
  		}
  	}
  	return true, ""
  }

  func ValidateModule() *runtime.Value {
  	return runtime.ObjectVal(map[string]*runtime.Value{
  		"isEmail": runtime.FuncVal(&runtime.Function{Name: "isEmail", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			return runtime.BoolVal(emailRegex.MatchString(args[0].ToString())), nil
  		}}),

  		"isUrl": runtime.FuncVal(&runtime.Function{Name: "isUrl", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			return runtime.BoolVal(urlRegex.MatchString(args[0].ToString())), nil
  		}}),

  		"isPhone": runtime.FuncVal(&runtime.Function{Name: "isPhone", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			return runtime.BoolVal(phoneRegex.MatchString(args[0].ToString())), nil
  		}}),

  		"isIPv4": runtime.FuncVal(&runtime.Function{Name: "isIPv4", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			return runtime.BoolVal(ipv4Regex.MatchString(args[0].ToString())), nil
  		}}),

  		"isIPv6": runtime.FuncVal(&runtime.Function{Name: "isIPv6", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			return runtime.BoolVal(ipv6Regex.MatchString(args[0].ToString())), nil
  		}}),

  		"isIP": runtime.FuncVal(&runtime.Function{Name: "isIP", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			s := args[0].ToString()
  			return runtime.BoolVal(ipv4Regex.MatchString(s) || ipv6Regex.MatchString(s)), nil
  		}}),

  		"isUUID": runtime.FuncVal(&runtime.Function{Name: "isUUID", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			return runtime.BoolVal(uuidRegex.MatchString(strings.ToLower(args[0].ToString()))), nil
  		}}),

  		"isAlpha": runtime.FuncVal(&runtime.Function{Name: "isAlpha", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			return runtime.BoolVal(alphaRegex.MatchString(args[0].ToString())), nil
  		}}),

  		"isAlphanumeric": runtime.FuncVal(&runtime.Function{Name: "isAlphanumeric", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			return runtime.BoolVal(alphaNumRegex.MatchString(args[0].ToString())), nil
  		}}),

  		"isHex": runtime.FuncVal(&runtime.Function{Name: "isHex", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			return runtime.BoolVal(hexRegex.MatchString(args[0].ToString())), nil
  		}}),

  		"isNumeric": runtime.FuncVal(&runtime.Function{Name: "isNumeric", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			if args[0].Tag == runtime.TypeNumber {
  				return runtime.True, nil
  			}
  			s := strings.TrimSpace(args[0].ToString())
  			if s == "" {
  				return runtime.False, nil
  			}
  			hasDot, hasE := false, false
  			for i, c := range s {
  				if c == '-' && i == 0 {
  					continue
  				}
  				if c == '.' && !hasDot {
  					hasDot = true
  					continue
  				}
  				if (c == 'e' || c == 'E') && !hasE {
  					hasE = true
  					continue
  				}
  				if c < '0' || c > '9' {
  					return runtime.False, nil
  				}
  			}
  			return runtime.True, nil
  		}}),

  		"isBase64": runtime.FuncVal(&runtime.Function{Name: "isBase64", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			s := args[0].ToString()
  			if len(s)%4 != 0 {
  				return runtime.False, nil
  			}
  			return runtime.BoolVal(base64Regex.MatchString(s)), nil
  		}}),

  		"isJSON": runtime.FuncVal(&runtime.Function{Name: "isJSON", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			s := strings.TrimSpace(args[0].ToString())
  			if len(s) == 0 {
  				return runtime.False, nil
  			}
  			_, err := parseJSON(s)
  			return runtime.BoolVal(err == nil), nil
  		}}),

  		"isCreditCard": runtime.FuncVal(&runtime.Function{Name: "isCreditCard", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			s := strings.ReplaceAll(args[0].ToString(), " ", "")
  			s = strings.ReplaceAll(s, "-", "")
  			if !creditCardRegex.MatchString(s) {
  				return runtime.False, nil
  			}
  			sum := 0
  			nDigits := len(s)
  			parity := nDigits % 2
  			for i, digit := range s {
  				d := int(digit - '0')
  				if i%2 == parity {
  					d *= 2
  					if d > 9 {
  						d -= 9
  					}
  				}
  				sum += d
  			}
  			return runtime.BoolVal(sum%10 == 0), nil
  		}}),

  		"isSlug": runtime.FuncVal(&runtime.Function{Name: "isSlug", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			return runtime.BoolVal(slugRegex.MatchString(args[0].ToString())), nil
  		}}),

  		"isDate": runtime.FuncVal(&runtime.Function{Name: "isDate", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			if args[0].Tag == runtime.TypeNumber {
  				return runtime.True, nil
  			}
  			dateRe := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
  			return runtime.BoolVal(dateRe.MatchString(args[0].ToString())), nil
  		}}),

  		"isStrongPassword": runtime.FuncVal(&runtime.Function{Name: "isStrongPassword", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			s := args[0].ToString()
  			if len(s) < 8 {
  				return runtime.False, nil
  			}
  			hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(s)
  			hasLower := regexp.MustCompile(`[a-z]`).MatchString(s)
  			hasDigit := regexp.MustCompile(`[0-9]`).MatchString(s)
  			hasSpecial := regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(s)
  			return runtime.BoolVal(hasUpper && hasLower && hasDigit && hasSpecial), nil
  		}}),

  		"schema": runtime.FuncVal(&runtime.Function{Name: "schema", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.ObjectVal(nil), nil
  			}
  			schemaDef := args[0]
  			return runtime.ObjectVal(map[string]*runtime.Value{
  				"validate": runtime.FuncVal(&runtime.Function{Name: "validate", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  					if len(a) == 0 {
  						return runtime.ObjectVal(map[string]*runtime.Value{"valid": runtime.False, "error": runtime.StringVal("no value")}), nil
  					}
  					ok, msg := validateSchema(a[0], schemaDef)
  					if ok {
  						return runtime.ObjectVal(map[string]*runtime.Value{"valid": runtime.True, "error": runtime.Null}), nil
  					}
  					return runtime.ObjectVal(map[string]*runtime.Value{"valid": runtime.False, "error": runtime.StringVal(msg)}), nil
  				}}),
  				"check": runtime.FuncVal(&runtime.Function{Name: "check", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  					if len(a) == 0 {
  						return runtime.False, nil
  					}
  					ok, _ := validateSchema(a[0], schemaDef)
  					return runtime.BoolVal(ok), nil
  				}}),
  				"assert": runtime.FuncVal(&runtime.Function{Name: "assert", Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  					if len(a) == 0 {
  						return runtime.Undefined, nil
  					}
  					ok, msg := validateSchema(a[0], schemaDef)
  					if !ok {
  						return runtime.Undefined, fmt.Errorf("%s", msg)
  					}
  					return a[0], nil
  				}}),
  			}), nil
  		}}),

  		"validate": runtime.FuncVal(&runtime.Function{Name: "validate", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) < 2 {
  				return runtime.ObjectVal(map[string]*runtime.Value{"valid": runtime.True, "errors": runtime.ArrayVal(nil)}), nil
  			}
  			val := args[0]
  			schema := args[1]
  			var errors []*runtime.Value
  			if schema.Tag == runtime.TypeObject {
  				for field, fieldSchema := range schema.ObjVal {
  					fieldVal := val.ObjVal[field]
  					if fieldVal == nil {
  						fieldVal = runtime.Undefined
  					}
  					if ok, msg := validateSchema(fieldVal, fieldSchema); !ok {
  						errors = append(errors, runtime.ObjectVal(map[string]*runtime.Value{
  							"field":   runtime.StringVal(field),
  							"message": runtime.StringVal(msg),
  						}))
  					}
  				}
  			}
  			return runtime.ObjectVal(map[string]*runtime.Value{
  				"valid":  runtime.BoolVal(len(errors) == 0),
  				"errors": runtime.ArrayVal(errors),
  			}), nil
  		}}),

  		"matches": runtime.FuncVal(&runtime.Function{Name: "matches", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) < 2 {
  				return runtime.False, nil
  			}
  			re, err := regexp.Compile(args[1].ToString())
  			if err != nil {
  				return runtime.False, nil
  			}
  			return runtime.BoolVal(re.MatchString(args[0].ToString())), nil
  		}}),

  		"length": runtime.FuncVal(&runtime.Function{Name: "length", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) < 3 {
  				return runtime.False, nil
  			}
  			s := args[0].ToString()
  			n := len(s)
  			min := int(args[1].ToNumber())
  			max := int(args[2].ToNumber())
  			return runtime.BoolVal(n >= min && n <= max), nil
  		}}),

  		"range": runtime.FuncVal(&runtime.Function{Name: "range", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) < 3 {
  				return runtime.False, nil
  			}
  			n := args[0].ToNumber()
  			min := args[1].ToNumber()
  			max := args[2].ToNumber()
  			return runtime.BoolVal(n >= min && n <= max), nil
  		}}),
  	})
  }
