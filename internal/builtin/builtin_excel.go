// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"fmt"
	"lunex/internal/runtime"

	"github.com/xuri/excelize/v2"
)

func ExcelModule() *runtime.Value {
	create := runtime.FuncVal(&runtime.Function{
		Name: "create",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			f := excelize.NewFile()
			return excelFileObj(f), nil
		},
	})

	open := runtime.FuncVal(&runtime.Function{
		Name: "open",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("open(path)")
			}
			f, err := excelize.OpenFile(args[0].ToString())
			if err != nil {
				return runtime.Null, err
			}
			return excelFileObj(f), nil
		},
	})

	columnName := runtime.FuncVal(&runtime.Function{
		Name: "columnName",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeNumber {
				return runtime.Null, fmt.Errorf("columnName(number)")
			}
			name, err := excelize.ColumnNumberToName(int(args[0].NumVal))
			if err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(name), nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"create":     create,
		"open":       open,
		"columnName": columnName,
	})
}

func excelFileObj(f *excelize.File) *runtime.Value {
	setCell := runtime.FuncVal(&runtime.Function{
		Name: "setCell",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				return runtime.Null, fmt.Errorf("setCell(sheet, cell, value)")
			}
			sheet := args[0].ToString()
			cell := args[1].ToString()
			var val interface{}
			switch args[2].Tag {
			case runtime.TypeNumber:
				val = args[2].NumVal
			case runtime.TypeBool:
				val = args[2].BoolVal
			case runtime.TypeNull, runtime.TypeUndefined:
				val = ""
			default:
				val = args[2].ToString()
			}
			err := f.SetCellValue(sheet, cell, val)
			return runtime.BoolVal(err == nil), err
		},
	})

	getCell := runtime.FuncVal(&runtime.Function{
		Name: "getCell",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("getCell(sheet, cell)")
			}
			val, err := f.GetCellValue(args[0].ToString(), args[1].ToString())
			if err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(val), nil
		},
	})

	getRow := runtime.FuncVal(&runtime.Function{
		Name: "getRow",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("getRow(sheet, rowIndex)")
			}
			sheet := args[0].ToString()
			row := int(args[1].NumVal)
			cols, err := f.GetCols(sheet)
			if err != nil || row <= 0 || row > len(cols) {
				return runtime.ArrayVal(nil), err
			}
			rowData := make([]*runtime.Value, 0)
			for _, col := range cols {
				if row-1 < len(col) {
					rowData = append(rowData, runtime.StringVal(col[row-1]))
				} else {
					rowData = append(rowData, runtime.StringVal(""))
				}
			}
			return runtime.ArrayVal(rowData), nil
		},
	})

	getRows := runtime.FuncVal(&runtime.Function{
		Name: "getRows",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), fmt.Errorf("getRows(sheet)")
			}
			rows, err := f.GetRows(args[0].ToString())
			if err != nil {
				return runtime.ArrayVal(nil), err
			}
			result := make([]*runtime.Value, len(rows))
			for i, row := range rows {
				cells := make([]*runtime.Value, len(row))
				for j, cell := range row {
					cells[j] = runtime.StringVal(cell)
				}
				result[i] = runtime.ArrayVal(cells)
			}
			return runtime.ArrayVal(result), nil
		},
	})

	getRowsAsObjects := runtime.FuncVal(&runtime.Function{
		Name: "getRowsAsObjects",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), fmt.Errorf("getRowsAsObjects(sheet)")
			}
			rows, err := f.GetRows(args[0].ToString())
			if err != nil || len(rows) == 0 {
				return runtime.ArrayVal(nil), err
			}
			headers := rows[0]
			result := make([]*runtime.Value, 0, len(rows)-1)
			for _, row := range rows[1:] {
				obj := make(map[string]*runtime.Value)
				for j, h := range headers {
					val := ""
					if j < len(row) {
						val = row[j]
					}
					obj[h] = runtime.StringVal(val)
				}
				result = append(result, runtime.ObjectVal(obj))
			}
			return runtime.ArrayVal(result), nil
		},
	})

	writeRow := runtime.FuncVal(&runtime.Function{
		Name: "writeRow",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				return runtime.Null, fmt.Errorf("writeRow(sheet, rowIndex, values)")
			}
			sheet := args[0].ToString()
			row := int(args[1].NumVal)
			if args[2].Tag != runtime.TypeArray {
				return runtime.Null, fmt.Errorf("values must be an array")
			}
			for col, v := range args[2].ArrVal {
				cell, _ := excelize.CoordinatesToCellName(col+1, row)
				var val interface{}
				if v.Tag == runtime.TypeNumber {
					val = v.NumVal
				} else if v.Tag == runtime.TypeBool {
					val = v.BoolVal
				} else {
					val = v.ToString()
				}
				f.SetCellValue(sheet, cell, val)
			}
			return runtime.True, nil
		},
	})

	newSheet := runtime.FuncVal(&runtime.Function{
		Name: "newSheet",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("newSheet(name)")
			}
			idx, err := f.NewSheet(args[0].ToString())
			if err != nil {
				return runtime.Null, err
			}
			return runtime.NumberVal(float64(idx)), nil
		},
	})

	deleteSheet := runtime.FuncVal(&runtime.Function{
		Name: "deleteSheet",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("deleteSheet(name)")
			}
			return runtime.BoolVal(f.DeleteSheet(args[0].ToString()) == nil), nil
		},
	})

	getSheets := runtime.FuncVal(&runtime.Function{
		Name: "getSheets",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			sheets := f.GetSheetList()
			arr := make([]*runtime.Value, len(sheets))
			for i, s := range sheets {
				arr[i] = runtime.StringVal(s)
			}
			return runtime.ArrayVal(arr), nil
		},
	})

	setStyle := runtime.FuncVal(&runtime.Function{
		Name: "setStyle",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				return runtime.Null, fmt.Errorf("setStyle(sheet, cell, styleJSON)")
			}
			style := &excelize.Style{}
			if args[2].Tag == runtime.TypeObject {
				opts := args[2].ObjVal
				if v, ok := opts["bold"]; ok && v.BoolVal {
					style.Font = &excelize.Font{Bold: true}
				}
				if v, ok := opts["fontSize"]; ok && v.Tag == runtime.TypeNumber {
					if style.Font == nil {
						style.Font = &excelize.Font{}
					}
					style.Font.Size = v.NumVal
				}
				if v, ok := opts["fontColor"]; ok {
					if style.Font == nil {
						style.Font = &excelize.Font{}
					}
					style.Font.Color = v.ToString()
				}
				if v, ok := opts["bgColor"]; ok {
					style.Fill = excelize.Fill{
						Type:    "pattern",
						Color:   []string{v.ToString()},
						Pattern: 1,
					}
				}
			}
			styleID, err := f.NewStyle(style)
			if err != nil {
				return runtime.Null, err
			}
			err = f.SetCellStyle(args[0].ToString(), args[1].ToString(), args[1].ToString(), styleID)
			return runtime.BoolVal(err == nil), err
		},
	})

	mergeCell := runtime.FuncVal(&runtime.Function{
		Name: "mergeCell",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				return runtime.Null, fmt.Errorf("mergeCell(sheet, start, end)")
			}
			err := f.MergeCell(args[0].ToString(), args[1].ToString(), args[2].ToString())
			return runtime.BoolVal(err == nil), err
		},
	})

	setColWidth := runtime.FuncVal(&runtime.Function{
		Name: "setColWidth",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 4 {
				return runtime.Null, fmt.Errorf("setColWidth(sheet, startCol, endCol, width)")
			}
			err := f.SetColWidth(args[0].ToString(), args[1].ToString(), args[2].ToString(), args[3].NumVal)
			return runtime.BoolVal(err == nil), err
		},
	})

	setFormula := runtime.FuncVal(&runtime.Function{
		Name: "setFormula",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				return runtime.Null, fmt.Errorf("setFormula(sheet, cell, formula)")
			}
			err := f.SetCellFormula(args[0].ToString(), args[1].ToString(), args[2].ToString())
			return runtime.BoolVal(err == nil), err
		},
	})

	save := runtime.FuncVal(&runtime.Function{
		Name: "save",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("save(path)")
			}
			err := f.SaveAs(args[0].ToString())
			return runtime.BoolVal(err == nil), err
		},
	})

	closeFile := runtime.FuncVal(&runtime.Function{
		Name: "close",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.Undefined, f.Close()
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"setCell":          setCell,
		"getCell":          getCell,
		"getRow":           getRow,
		"getRows":          getRows,
		"getRowsAsObjects": getRowsAsObjects,
		"writeRow":         writeRow,
		"newSheet":         newSheet,
		"deleteSheet":      deleteSheet,
		"getSheets":        getSheets,
		"setStyle":         setStyle,
		"mergeCell":        mergeCell,
		"setColWidth":      setColWidth,
		"setFormula":       setFormula,
		"save":             save,
		"close":            closeFile,
	})
}
