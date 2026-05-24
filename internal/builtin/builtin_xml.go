// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

  import (
  	"bytes"
  	"encoding/xml"
  	"fmt"
  	"lunex/internal/runtime"
  	"os"
  	"strings"
  )

  func XMLModule() *runtime.Value {
  	parse := runtime.FuncVal(&runtime.Function{
  		Name: "parse",
  		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.Null, fmt.Errorf("xml.parse(xmlString)")
  			}
  			doc, err := xmlStringToValue(args[0].ToString())
  			if err != nil {
  				return runtime.Null, err
  			}
  			return doc, nil
  		},
  	})

  	stringify := runtime.FuncVal(&runtime.Function{
  		Name: "stringify",
  		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.StringVal(""), fmt.Errorf("xml.stringify(object, rootTag?, options?)")
  			}
  			rootTag := "root"
  			if len(args) > 1 && args[1] != nil && args[1].Tag == runtime.TypeString {
  				rootTag = args[1].StrVal
  			}
  			pretty := false
  			if len(args) > 2 && args[2] != nil && args[2].Tag == runtime.TypeObject {
  				if v, ok := args[2].ObjVal["pretty"]; ok {
  					pretty = v.BoolVal
  				}
  			}
  			var buf bytes.Buffer
  			buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
  			if pretty {
  				buf.WriteByte('\n')
  			}
  			writeXMLValue(&buf, args[0], rootTag, 0, pretty)
  			return runtime.StringVal(buf.String()), nil
  		},
  	})

  	validate := runtime.FuncVal(&runtime.Function{
  		Name: "validate",
  		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.False, nil
  			}
  			dec := xml.NewDecoder(strings.NewReader(args[0].ToString()))
  			for {
  				_, err := dec.Token()
  				if err != nil {
  					if err.Error() == "EOF" {
  						return runtime.True, nil
  					}
  					return runtime.False, nil
  				}
  			}
  		},
  	})

  	query := runtime.FuncVal(&runtime.Function{
  		Name: "query",
  		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) < 2 {
  				return runtime.ArrayVal(nil), fmt.Errorf("xml.query(xmlString, path)")
  			}
  			doc, err := xmlStringToValue(args[0].ToString())
  			if err != nil {
  				return runtime.ArrayVal(nil), err
  			}
  			parts := strings.Split(strings.Trim(args[1].ToString(), "/"), "/")
  			results := xmlQuery(doc, parts)
  			return runtime.ArrayVal(results), nil
  		},
  	})

  	readFile := runtime.FuncVal(&runtime.Function{
  		Name: "readFile",
  		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.Null, fmt.Errorf("xml.readFile(path)")
  			}
  			data, err := os.ReadFile(args[0].ToString())
  			if err != nil {
  				return runtime.Null, err
  			}
  			return xmlStringToValue(string(data))
  		},
  	})

  	writeFile := runtime.FuncVal(&runtime.Function{
  		Name: "writeFile",
  		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) < 2 {
  				return runtime.False, fmt.Errorf("xml.writeFile(path, object, rootTag?, options?)")
  			}
  			rootTag := "root"
  			if len(args) > 2 && args[2] != nil && args[2].Tag == runtime.TypeString {
  				rootTag = args[2].StrVal
  			}
  			pretty := true
  			var buf bytes.Buffer
  			buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>\n`)
  			writeXMLValue(&buf, args[1], rootTag, 0, pretty)
  			err := os.WriteFile(args[0].ToString(), buf.Bytes(), 0644)
  			if err != nil {
  				return runtime.False, err
  			}
  			return runtime.True, nil
  		},
  	})

  	getAttribute := runtime.FuncVal(&runtime.Function{
  		Name: "getAttribute",
  		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) < 2 {
  				return runtime.Null, nil
  			}
  			obj := args[0]
  			name := args[1].ToString()
  			if obj == nil || obj.Tag != runtime.TypeObject {
  				return runtime.Null, nil
  			}
  			attrs, ok := obj.ObjVal["@attributes"]
  			if !ok || attrs == nil || attrs.Tag != runtime.TypeObject {
  				return runtime.Null, nil
  			}
  			if v, ok := attrs.ObjVal[name]; ok {
  				return v, nil
  			}
  			return runtime.Null, nil
  		},
  	})

  	getText := runtime.FuncVal(&runtime.Function{
  		Name: "getText",
  		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
  			if len(args) == 0 {
  				return runtime.StringVal(""), nil
  			}
  			obj := args[0]
  			if obj == nil {
  				return runtime.StringVal(""), nil
  			}
  			if obj.Tag == runtime.TypeString {
  				return obj, nil
  			}
  			if obj.Tag == runtime.TypeObject {
  				if v, ok := obj.ObjVal["#text"]; ok && v != nil {
  					return runtime.StringVal(v.ToString()), nil
  				}
  			}
  			return runtime.StringVal(""), nil
  		},
  	})

  	return runtime.ObjectVal(map[string]*runtime.Value{
  		"parse":        parse,
  		"stringify":    stringify,
  		"validate":     validate,
  		"query":        query,
  		"readFile":     readFile,
  		"writeFile":    writeFile,
  		"getAttribute": getAttribute,
  		"getText":      getText,
  	})
  }

  func xmlStringToValue(s string) (*runtime.Value, error) {
  	dec := xml.NewDecoder(strings.NewReader(s))
  	var stack []*xmlNode
  	root := &xmlNode{tag: "__root__", children: make(map[string][]*xmlNode)}
  	stack = append(stack, root)

  	for {
  		tok, err := dec.Token()
  		if err != nil {
  			break
  		}
  		switch t := tok.(type) {
  		case xml.StartElement:
  			node := &xmlNode{
  				tag:      t.Name.Local,
  				attrs:    make(map[string]string),
  				children: make(map[string][]*xmlNode),
  			}
  			for _, attr := range t.Attr {
  				node.attrs[attr.Name.Local] = attr.Value
  			}
  			stack = append(stack, node)
  		case xml.EndElement:
  			if len(stack) <= 1 {
  				continue
  			}
  			node := stack[len(stack)-1]
  			stack = stack[:len(stack)-1]
  			parent := stack[len(stack)-1]
  			parent.children[node.tag] = append(parent.children[node.tag], node)
  		case xml.CharData:
  			text := strings.TrimSpace(string(t))
  			if text != "" && len(stack) > 0 {
  				stack[len(stack)-1].text += text
  			}
  		}
  	}

  	if len(root.children) == 0 {
  		return runtime.Null, fmt.Errorf("invalid or empty XML")
  	}
  	for tag, children := range root.children {
  		if len(children) == 1 {
  			return xmlNodeToValue(children[0], tag), nil
  		}
  		arr := make([]*runtime.Value, len(children))
  		for i, c := range children {
  			arr[i] = xmlNodeToValue(c, tag)
  		}
  		return runtime.ArrayVal(arr), nil
  	}
  	return runtime.Null, nil
  }

  type xmlNode struct {
  	tag      string
  	text     string
  	attrs    map[string]string
  	children map[string][]*xmlNode
  }

  func xmlNodeToValue(node *xmlNode, _ string) *runtime.Value {
  	obj := make(map[string]*runtime.Value)
  	obj["#tag"] = runtime.StringVal(node.tag)
  	if node.text != "" {
  		obj["#text"] = runtime.StringVal(node.text)
  	}
  	if len(node.attrs) > 0 {
  		attrObj := make(map[string]*runtime.Value)
  		for k, v := range node.attrs {
  			attrObj[k] = runtime.StringVal(v)
  		}
  		obj["@attributes"] = runtime.ObjectVal(attrObj)
  	}
  	for tag, children := range node.children {
  		if len(children) == 1 {
  			obj[tag] = xmlNodeToValue(children[0], tag)
  		} else {
  			arr := make([]*runtime.Value, len(children))
  			for i, c := range children {
  				arr[i] = xmlNodeToValue(c, tag)
  			}
  			obj[tag] = runtime.ArrayVal(arr)
  		}
  	}
  	return runtime.ObjectVal(obj)
  }

  func xmlQuery(v *runtime.Value, parts []string) []*runtime.Value {
  	if v == nil || len(parts) == 0 {
  		return nil
  	}
  	key := parts[0]
  	if key == "" {
  		return nil
  	}
  	if v.Tag == runtime.TypeObject {
  		child, ok := v.ObjVal[key]
  		if !ok {
  			return nil
  		}
  		if len(parts) == 1 {
  			if child.Tag == runtime.TypeArray {
  				return child.ArrVal
  			}
  			return []*runtime.Value{child}
  		}
  		return xmlQuery(child, parts[1:])
  	}
  	if v.Tag == runtime.TypeArray {
  		var results []*runtime.Value
  		for _, el := range v.ArrVal {
  			results = append(results, xmlQuery(el, parts)...)
  		}
  		return results
  	}
  	return nil
  }

  func writeXMLValue(buf *bytes.Buffer, v *runtime.Value, tag string, depth int, pretty bool) {
  	indent := ""
  	if pretty {
  		indent = strings.Repeat("  ", depth)
  	}
  	newline := ""
  	if pretty {
  		newline = "\n"
  	}

  	if v == nil || v.Tag == runtime.TypeNull || v.Tag == runtime.TypeUndefined {
  		return
  	}

  	safeTag := xmlEscapeTag(tag)

  	if v.Tag == runtime.TypeArray {
  		for _, el := range v.ArrVal {
  			writeXMLValue(buf, el, tag, depth, pretty)
  		}
  		return
  	}

  	if v.Tag == runtime.TypeObject {
  		attrs := ""
  		if attrVal, ok := v.ObjVal["@attributes"]; ok && attrVal != nil && attrVal.Tag == runtime.TypeObject {
  			var sb strings.Builder
  			for k, val := range attrVal.ObjVal {
  				sb.WriteString(fmt.Sprintf(` %s="%s"`, xmlEscapeTag(k), xmlEscapeText(val.ToString())))
  			}
  			attrs = sb.String()
  		}
  		buf.WriteString(indent + "<" + safeTag + attrs + ">" + newline)
  		if textVal, ok := v.ObjVal["#text"]; ok && textVal != nil {
  			if pretty {
  				buf.WriteString(strings.Repeat("  ", depth+1))
  			}
  			buf.WriteString(xmlEscapeText(textVal.ToString()))
  			buf.WriteString(newline)
  		}
  		for k, child := range v.ObjVal {
  			if k == "@attributes" || k == "#text" || k == "#tag" {
  				continue
  			}
  			writeXMLValue(buf, child, k, depth+1, pretty)
  		}
  		buf.WriteString(indent + "</" + safeTag + ">" + newline)
  		return
  	}

  	buf.WriteString(indent + "<" + safeTag + ">" + xmlEscapeText(v.ToString()) + "</" + safeTag + ">" + newline)
  }

  func xmlEscapeTag(s string) string {
  	s = strings.Map(func(r rune) rune {
  		if r == '<' || r == '>' || r == '&' || r == '"' || r == '\'' || r == ' ' {
  			return '_'
  		}
  		return r
  	}, s)
  	return s
  }

  func xmlEscapeText(s string) string {
  	s = strings.ReplaceAll(s, "&", "&amp;")
  	s = strings.ReplaceAll(s, "<", "&lt;")
  	s = strings.ReplaceAll(s, ">", "&gt;")
  	s = strings.ReplaceAll(s, "\"", "&quot;")
  	return s
  }
