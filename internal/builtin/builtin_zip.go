// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"archive/zip"
	"bytes"
	"io"
	"lunex/internal/runtime"
	"os"
	"path/filepath"
	"strings"
)

func ZipModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"create": runtime.FuncVal(&runtime.Function{
			Name: "create",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) < 2 {
					return runtime.Null, nil
				}
				outputPath := args[0].ToString()
				filesVal := args[1]
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				if filesVal.Tag == runtime.TypeArray {
					for _, item := range filesVal.ArrVal {
						if item == nil || item.Tag != runtime.TypeObject {
							continue
						}
						nameV, hasName := item.ObjVal["name"]
						contV, hasCont := item.ObjVal["content"]
						if !hasName || !hasCont {
							continue
						}
						f, err := w.Create(nameV.ToString())
						if err != nil {
							continue
						}
						f.Write([]byte(contV.ToString()))
					}
				}
				if err := w.Close(); err != nil {
					return runtime.Null, err
				}
				if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
					return runtime.Null, err
				}
				return runtime.NumberVal(float64(buf.Len())), nil
			},
		}),

		"extract": runtime.FuncVal(&runtime.Function{
			Name: "extract",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) < 2 {
					return runtime.Null, nil
				}
				zipPath := args[0].ToString()
				destDir := args[1].ToString()
				r, err := zip.OpenReader(zipPath)
				if err != nil {
					return runtime.Null, err
				}
				defer r.Close()
				extracted := 0
				for _, f := range r.File {
					outPath := filepath.Join(destDir, f.Name)
					if f.FileInfo().IsDir() {
						os.MkdirAll(outPath, 0755)
						continue
					}
					os.MkdirAll(filepath.Dir(outPath), 0755)
					rc, err := f.Open()
					if err != nil {
						continue
					}
					data, err := io.ReadAll(rc)
					rc.Close()
					if err != nil {
						continue
					}
					os.WriteFile(outPath, data, 0644)
					extracted++
				}
				return runtime.NumberValInt(int64(extracted)), nil
			},
		}),

		"list": runtime.FuncVal(&runtime.Function{
			Name: "list",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) < 1 {
					return runtime.ArrayVal(nil), nil
				}
				zipPath := args[0].ToString()
				r, err := zip.OpenReader(zipPath)
				if err != nil {
					return runtime.ArrayVal(nil), err
				}
				defer r.Close()
				entries := make([]*runtime.Value, 0, len(r.File))
				for _, f := range r.File {
					entry := runtime.ObjectVal(map[string]*runtime.Value{
						"name":      runtime.StringVal(f.Name),
						"size":      runtime.NumberValInt(int64(f.UncompressedSize64)),
						"compressed": runtime.NumberValInt(int64(f.CompressedSize64)),
						"isDir":     runtime.BoolVal(f.FileInfo().IsDir()),
					})
					entries = append(entries, entry)
				}
				return runtime.ArrayVal(entries), nil
			},
		}),

		"read": runtime.FuncVal(&runtime.Function{
			Name: "read",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) < 2 {
					return runtime.Null, nil
				}
				zipPath := args[0].ToString()
				entryName := args[1].ToString()
				r, err := zip.OpenReader(zipPath)
				if err != nil {
					return runtime.Null, err
				}
				defer r.Close()
				for _, f := range r.File {
					if f.Name == entryName {
						rc, err := f.Open()
						if err != nil {
							return runtime.Null, err
						}
						data, err := io.ReadAll(rc)
						rc.Close()
						if err != nil {
							return runtime.Null, err
						}
						return runtime.StringVal(string(data)), nil
					}
				}
				return runtime.Null, nil
			},
		}),

		"addFile": runtime.FuncVal(&runtime.Function{
			Name: "addFile",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) < 2 {
					return runtime.False, nil
				}
				zipPath := args[0].ToString()
				filePath := args[1].ToString()
				entryName := filepath.Base(filePath)
				if len(args) >= 3 {
					entryName = args[2].ToString()
				}
				existing, _ := os.ReadFile(zipPath)
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				if len(existing) > 0 {
					r, err := zip.NewReader(bytes.NewReader(existing), int64(len(existing)))
					if err == nil {
						for _, f := range r.File {
							if f.Name == entryName {
								continue
							}
							fw, err := w.CreateHeader(&f.FileHeader)
							if err != nil {
								continue
							}
							rc, err := f.Open()
							if err != nil {
								continue
							}
							io.Copy(fw, rc)
							rc.Close()
						}
					}
				}
				data, err := os.ReadFile(filePath)
				if err != nil {
					w.Close()
					return runtime.False, err
				}
				fw, err := w.Create(entryName)
				if err != nil {
					w.Close()
					return runtime.False, err
				}
				fw.Write(data)
				w.Close()
				os.WriteFile(zipPath, buf.Bytes(), 0644)
				return runtime.True, nil
			},
		}),

		"packDir": runtime.FuncVal(&runtime.Function{
			Name: "packDir",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) < 2 {
					return runtime.Null, nil
				}
				srcDir := args[0].ToString()
				outputPath := args[1].ToString()
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				count := 0
				filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
					if err != nil || d.IsDir() {
						return nil
					}
					rel, err := filepath.Rel(srcDir, path)
					if err != nil {
						return nil
					}
					rel = strings.ReplaceAll(rel, string(os.PathSeparator), "/")
					data, err := os.ReadFile(path)
					if err != nil {
						return nil
					}
					fw, err := w.Create(rel)
					if err != nil {
						return nil
					}
					fw.Write(data)
					count++
					return nil
				})
				w.Close()
				os.WriteFile(outputPath, buf.Bytes(), 0644)
				return runtime.NumberValInt(int64(count)), nil
			},
		}),

		"fromBytes": runtime.FuncVal(&runtime.Function{
			Name: "fromBytes",
			Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				if len(args) < 1 {
					return runtime.ArrayVal(nil), nil
				}
				raw := []byte(args[0].ToString())
				r, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
				if err != nil {
					return runtime.ArrayVal(nil), err
				}
				entries := make([]*runtime.Value, 0, len(r.File))
				for _, f := range r.File {
					rc, err := f.Open()
					if err != nil {
						continue
					}
					data, _ := io.ReadAll(rc)
					rc.Close()
					entry := runtime.ObjectVal(map[string]*runtime.Value{
						"name":    runtime.StringVal(f.Name),
						"content": runtime.StringVal(string(data)),
					})
					entries = append(entries, entry)
				}
				return runtime.ArrayVal(entries), nil
			},
		}),
	})
}
