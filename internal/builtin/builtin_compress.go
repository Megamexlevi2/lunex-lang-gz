package builtin

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"
	"lunex/internal/runtime"
)

func CompressModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"gzip": runtime.FuncVal(&runtime.Function{Name: "gzip", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			data := toBytes(args[0])
			level := gzip.DefaultCompression
			if len(args) > 1 {
				level = int(args[1].ToNumber())
			}
			var buf bytes.Buffer
			w, err := gzip.NewWriterLevel(&buf, level)
			if err != nil {
				return nil, fmt.Errorf("compress.gzip: %v", err)
			}
			if _, err := w.Write(data); err != nil {
				return nil, fmt.Errorf("compress.gzip: %v", err)
			}
			if err := w.Close(); err != nil {
				return nil, fmt.Errorf("compress.gzip: %v", err)
			}
			return runtime.StringVal(base64.StdEncoding.EncodeToString(buf.Bytes())), nil
		}}),

		"gunzip": runtime.FuncVal(&runtime.Function{Name: "gunzip", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			data, err := fromBytesOrBase64(args[0])
			if err != nil {
				return nil, fmt.Errorf("compress.gunzip: %v", err)
			}
			r, err := gzip.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("compress.gunzip: %v", err)
			}
			defer r.Close()
			out, err := io.ReadAll(r)
			if err != nil {
				return nil, fmt.Errorf("compress.gunzip: %v", err)
			}
			return runtime.StringVal(string(out)), nil
		}}),

		"gzipBytes": runtime.FuncVal(&runtime.Function{Name: "gzipBytes", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			data := toBytes(args[0])
			level := gzip.DefaultCompression
			if len(args) > 1 {
				level = int(args[1].ToNumber())
			}
			var buf bytes.Buffer
			w, err := gzip.NewWriterLevel(&buf, level)
			if err != nil {
				return nil, fmt.Errorf("compress.gzipBytes: %v", err)
			}
			if _, err := w.Write(data); err != nil {
				return nil, fmt.Errorf("compress.gzipBytes: %v", err)
			}
			if err := w.Close(); err != nil {
				return nil, fmt.Errorf("compress.gzipBytes: %v", err)
			}
			result := buf.Bytes()
			out := make([]*runtime.Value, len(result))
			for i, b := range result {
				out[i] = runtime.NumberVal(float64(b))
			}
			return runtime.ArrayVal(out), nil
		}}),

		"gunzipBytes": runtime.FuncVal(&runtime.Function{Name: "gunzipBytes", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), nil
			}
			var data []byte
			if args[0].Tag == runtime.TypeArray {
				data = make([]byte, len(args[0].ArrVal))
				for i, v := range args[0].ArrVal {
					if v != nil {
						data[i] = byte(int(v.ToNumber()))
					}
				}
			} else {
				var err error
				data, err = fromBytesOrBase64(args[0])
				if err != nil {
					return nil, fmt.Errorf("compress.gunzipBytes: %v", err)
				}
			}
			r, err := gzip.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("compress.gunzipBytes: %v", err)
			}
			defer r.Close()
			out, err := io.ReadAll(r)
			if err != nil {
				return nil, fmt.Errorf("compress.gunzipBytes: %v", err)
			}
			result := make([]*runtime.Value, len(out))
			for i, b := range out {
				result[i] = runtime.NumberVal(float64(b))
			}
			return runtime.ArrayVal(result), nil
		}}),

		"deflate": runtime.FuncVal(&runtime.Function{Name: "deflate", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			data := toBytes(args[0])
			level := flate.DefaultCompression
			if len(args) > 1 {
				level = int(args[1].ToNumber())
			}
			var buf bytes.Buffer
			w, err := flate.NewWriter(&buf, level)
			if err != nil {
				return nil, fmt.Errorf("compress.deflate: %v", err)
			}
			if _, err := w.Write(data); err != nil {
				return nil, fmt.Errorf("compress.deflate: %v", err)
			}
			if err := w.Close(); err != nil {
				return nil, fmt.Errorf("compress.deflate: %v", err)
			}
			return runtime.StringVal(base64.StdEncoding.EncodeToString(buf.Bytes())), nil
		}}),

		"inflate": runtime.FuncVal(&runtime.Function{Name: "inflate", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			data, err := fromBytesOrBase64(args[0])
			if err != nil {
				return nil, fmt.Errorf("compress.inflate: %v", err)
			}
			r := flate.NewReader(bytes.NewReader(data))
			defer r.Close()
			out, err := io.ReadAll(r)
			if err != nil {
				return nil, fmt.Errorf("compress.inflate: %v", err)
			}
			return runtime.StringVal(string(out)), nil
		}}),

		"zlib": runtime.FuncVal(&runtime.Function{Name: "zlib", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			data := toBytes(args[0])
			level := zlib.DefaultCompression
			if len(args) > 1 {
				level = int(args[1].ToNumber())
			}
			var buf bytes.Buffer
			w, err := zlib.NewWriterLevel(&buf, level)
			if err != nil {
				return nil, fmt.Errorf("compress.zlib: %v", err)
			}
			if _, err := w.Write(data); err != nil {
				return nil, fmt.Errorf("compress.zlib: %v", err)
			}
			if err := w.Close(); err != nil {
				return nil, fmt.Errorf("compress.zlib: %v", err)
			}
			return runtime.StringVal(base64.StdEncoding.EncodeToString(buf.Bytes())), nil
		}}),

		"unzlib": runtime.FuncVal(&runtime.Function{Name: "unzlib", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			data, err := fromBytesOrBase64(args[0])
			if err != nil {
				return nil, fmt.Errorf("compress.unzlib: %v", err)
			}
			r, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("compress.unzlib: %v", err)
			}
			defer r.Close()
			out, err := io.ReadAll(r)
			if err != nil {
				return nil, fmt.Errorf("compress.unzlib: %v", err)
			}
			return runtime.StringVal(string(out)), nil
		}}),

		"ratio": runtime.FuncVal(&runtime.Function{Name: "ratio", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(0), nil
			}
			data := toBytes(args[0])
			if len(data) == 0 {
				return runtime.NumberVal(0), nil
			}
			var buf bytes.Buffer
			w, err := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
			if err != nil {
				return runtime.NumberVal(0), nil
			}
			w.Write(data)
			w.Close()
			return runtime.NumberVal(float64(buf.Len()) / float64(len(data))), nil
		}}),

		"level": runtime.ObjectVal(map[string]*runtime.Value{
			"none":    runtime.NumberVal(float64(gzip.NoCompression)),
			"fast":    runtime.NumberVal(float64(gzip.BestSpeed)),
			"default": runtime.NumberVal(float64(gzip.DefaultCompression)),
			"best":    runtime.NumberVal(float64(gzip.BestCompression)),
		}),
	})
}

func toBytes(v *runtime.Value) []byte {
	if v == nil {
		return nil
	}
	if v.Tag == runtime.TypeArray {
		data := make([]byte, len(v.ArrVal))
		for i, el := range v.ArrVal {
			if el != nil {
				data[i] = byte(int(el.ToNumber()))
			}
		}
		return data
	}
	return []byte(v.ToString())
}

func fromBytesOrBase64(v *runtime.Value) ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("nil input")
	}
	if v.Tag == runtime.TypeArray {
		data := make([]byte, len(v.ArrVal))
		for i, el := range v.ArrVal {
			if el != nil {
				data[i] = byte(int(el.ToNumber()))
			}
		}
		return data, nil
	}
	s := v.ToString()
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return []byte(s), nil
	}
	return data, nil
}
