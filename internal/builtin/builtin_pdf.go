// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
        "fmt"
        "lunex/internal/runtime"

        "github.com/jung-kurt/gofpdf"
)

func PDFModule() *runtime.Value {
        create := runtime.FuncVal(&runtime.Function{
                Name: "create",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        orientation := "P"
                        unit := "mm"
                        size := "A4"
                        if len(args) > 0 && args[0].Tag == runtime.TypeObject {
                                opts := args[0].ObjVal
                                if v, ok := opts["orientation"]; ok {
                                        o := v.ToString()
                                        if o == "landscape" || o == "L" {
                                                orientation = "L"
                                        }
                                }
                                if v, ok := opts["unit"]; ok {
                                        unit = v.ToString()
                                }
                                if v, ok := opts["size"]; ok {
                                        size = v.ToString()
                                }
                        }
                        f := gofpdf.New(orientation, unit, size, "")
                        return pdfDocObj(f), nil
                },
        })

        return runtime.ObjectVal(map[string]*runtime.Value{
                "create": create,
        })
}

func pdfDocObj(f *gofpdf.Fpdf) *runtime.Value {
        addPage := runtime.FuncVal(&runtime.Function{
                Name: "addPage",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        f.AddPage()
                        return runtime.Undefined, nil
                },
        })

        setFont := runtime.FuncVal(&runtime.Function{
                Name: "setFont",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        family := "Arial"
                        style := ""
                        size := 12.0
                        if len(args) > 0 {
                                family = args[0].ToString()
                        }
                        if len(args) > 1 {
                                style = args[1].ToString()
                        }
                        if len(args) > 2 && args[2].Tag == runtime.TypeNumber {
                                size = args[2].NumVal
                        }
                        f.SetFont(family, style, size)
                        return runtime.Undefined, nil
                },
        })

        setFontSize := runtime.FuncVal(&runtime.Function{
                Name: "setFontSize",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) > 0 && args[0].Tag == runtime.TypeNumber {
                                f.SetFontSize(args[0].NumVal)
                        }
                        return runtime.Undefined, nil
                },
        })

        setTextColor := runtime.FuncVal(&runtime.Function{
                Name: "setTextColor",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        r, g, b := 0, 0, 0
                        if len(args) >= 3 {
                                r = int(args[0].NumVal)
                                g = int(args[1].NumVal)
                                b = int(args[2].NumVal)
                        }
                        f.SetTextColor(r, g, b)
                        return runtime.Undefined, nil
                },
        })

        setFillColor := runtime.FuncVal(&runtime.Function{
                Name: "setFillColor",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) >= 3 {
                                f.SetFillColor(int(args[0].NumVal), int(args[1].NumVal), int(args[2].NumVal))
                        }
                        return runtime.Undefined, nil
                },
        })

        setDrawColor := runtime.FuncVal(&runtime.Function{
                Name: "setDrawColor",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) >= 3 {
                                f.SetDrawColor(int(args[0].NumVal), int(args[1].NumVal), int(args[2].NumVal))
                        }
                        return runtime.Undefined, nil
                },
        })

        cell := runtime.FuncVal(&runtime.Function{
                Name: "cell",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        w := 0.0
                        h := 10.0
                        text := ""
                        if len(args) > 0 && args[0].Tag == runtime.TypeNumber {
                                w = args[0].NumVal
                        }
                        if len(args) > 1 && args[1].Tag == runtime.TypeNumber {
                                h = args[1].NumVal
                        }
                        if len(args) > 2 {
                                text = args[2].ToString()
                        }
                        border := ""
                        align := "L"
                        fill := false
                        newLine := 0
                        if len(args) > 3 && args[3].Tag == runtime.TypeObject {
                                opts := args[3].ObjVal
                                if v, ok := opts["border"]; ok {
                                        border = v.ToString()
                                }
                                if v, ok := opts["align"]; ok {
                                        align = v.ToString()
                                }
                                if v, ok := opts["fill"]; ok {
                                        fill = v.BoolVal
                                }
                                if v, ok := opts["newLine"]; ok && v.Tag == runtime.TypeNumber {
                                        newLine = int(v.NumVal)
                                }
                        }
                        f.CellFormat(w, h, text, border, newLine, align, fill, 0, "")
                        return runtime.Undefined, nil
                },
        })

        multiCell := runtime.FuncVal(&runtime.Function{
                Name: "multiCell",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        w := 0.0
                        h := 10.0
                        text := ""
                        if len(args) > 0 && args[0].Tag == runtime.TypeNumber {
                                w = args[0].NumVal
                        }
                        if len(args) > 1 && args[1].Tag == runtime.TypeNumber {
                                h = args[1].NumVal
                        }
                        if len(args) > 2 {
                                text = args[2].ToString()
                        }
                        f.MultiCell(w, h, text, "", "", false)
                        return runtime.Undefined, nil
                },
        })

        line := runtime.FuncVal(&runtime.Function{
                Name: "line",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) < 4 {
                                return runtime.Null, fmt.Errorf("line(x1, y1, x2, y2)")
                        }
                        f.Line(args[0].NumVal, args[1].NumVal, args[2].NumVal, args[3].NumVal)
                        return runtime.Undefined, nil
                },
        })

        rect := runtime.FuncVal(&runtime.Function{
                Name: "rect",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) < 4 {
                                return runtime.Null, fmt.Errorf("rect(x, y, w, h, style?)")
                        }
                        style := "D"
                        if len(args) > 4 {
                                style = args[4].ToString()
                        }
                        f.Rect(args[0].NumVal, args[1].NumVal, args[2].NumVal, args[3].NumVal, style)
                        return runtime.Undefined, nil
                },
        })

        setXY := runtime.FuncVal(&runtime.Function{
                Name: "setXY",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) >= 2 {
                                f.SetXY(args[0].NumVal, args[1].NumVal)
                        }
                        return runtime.Undefined, nil
                },
        })

        setX := runtime.FuncVal(&runtime.Function{
                Name: "setX",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) > 0 {
                                f.SetX(args[0].NumVal)
                        }
                        return runtime.Undefined, nil
                },
        })

        setY := runtime.FuncVal(&runtime.Function{
                Name: "setY",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) > 0 {
                                f.SetY(args[0].NumVal)
                        }
                        return runtime.Undefined, nil
                },
        })

        getX := runtime.FuncVal(&runtime.Function{
                Name: "getX",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.NumberVal(f.GetX()), nil
                },
        })

        getY := runtime.FuncVal(&runtime.Function{
                Name: "getY",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.NumberVal(f.GetY()), nil
                },
        })

        setMargins := runtime.FuncVal(&runtime.Function{
                Name: "setMargins",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        l, t, r := 10.0, 10.0, 10.0
                        if len(args) >= 3 {
                                l = args[0].NumVal
                                t = args[1].NumVal
                                r = args[2].NumVal
                        }
                        f.SetMargins(l, t, r)
                        return runtime.Undefined, nil
                },
        })

        setTitle := runtime.FuncVal(&runtime.Function{
                Name: "setTitle",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) > 0 {
                                f.SetTitle(args[0].ToString(), true)
                        }
                        return runtime.Undefined, nil
                },
        })

        setAuthor := runtime.FuncVal(&runtime.Function{
                Name: "setAuthor",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) > 0 {
                                f.SetAuthor(args[0].ToString(), true)
                        }
                        return runtime.Undefined, nil
                },
        })

        image := runtime.FuncVal(&runtime.Function{
                Name: "image",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) < 5 {
                                return runtime.Null, fmt.Errorf("image(path, x, y, w, h)")
                        }
                        opts := &gofpdf.ImageOptions{ImageType: ""}
                        f.ImageOptions(args[0].ToString(), args[1].NumVal, args[2].NumVal, args[3].NumVal, args[4].NumVal, false, *opts, 0, "")
                        return runtime.Undefined, nil
                },
        })

        setLineWidth := runtime.FuncVal(&runtime.Function{
                Name: "setLineWidth",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) > 0 {
                                f.SetLineWidth(args[0].NumVal)
                        }
                        return runtime.Undefined, nil
                },
        })

        ln := runtime.FuncVal(&runtime.Function{
                Name: "ln",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        h := -1.0
                        if len(args) > 0 && args[0].Tag == runtime.TypeNumber {
                                h = args[0].NumVal
                        }
                        f.Ln(h)
                        return runtime.Undefined, nil
                },
        })

        pageSize := runtime.FuncVal(&runtime.Function{
                Name: "pageSize",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        w, h := f.GetPageSize()
                        return runtime.ObjectVal(map[string]*runtime.Value{
                                "width":  runtime.NumberVal(w),
                                "height": runtime.NumberVal(h),
                        }), nil
                },
        })

        pageCount := runtime.FuncVal(&runtime.Function{
                Name: "pageCount",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.NumberVal(float64(f.PageCount())), nil
                },
        })

        save := runtime.FuncVal(&runtime.Function{
                Name: "save",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        if len(args) == 0 {
                                return runtime.Null, fmt.Errorf("save(path)")
                        }
                        err := f.OutputFileAndClose(args[0].ToString())
                        return runtime.BoolVal(err == nil), err
                },
        })

        ok := runtime.FuncVal(&runtime.Function{
                Name: "ok",
                Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
                        return runtime.BoolVal(f.Ok()), nil
                },
        })

        return runtime.ObjectVal(map[string]*runtime.Value{
                "addPage":      addPage,
                "setFont":      setFont,
                "setFontSize":  setFontSize,
                "setTextColor": setTextColor,
                "setFillColor": setFillColor,
                "setDrawColor": setDrawColor,
                "cell":         cell,
                "multiCell":    multiCell,
                "line":         line,
                "rect":         rect,
                "setXY":        setXY,
                "setX":         setX,
                "setY":         setY,
                "getX":         getX,
                "getY":         getY,
                "setMargins":   setMargins,
                "setTitle":     setTitle,
                "setAuthor":    setAuthor,
                "image":        image,
                "setLineWidth": setLineWidth,
                "ln":           ln,
                "pageSize":     pageSize,
                "pageCount":    pageCount,
                "save":         save,
                "ok":           ok,
        })
}
