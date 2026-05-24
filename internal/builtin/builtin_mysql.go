// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"database/sql"
	"fmt"
	"lunex/internal/runtime"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

var mysqlPools sync.Map

func mysqlColToNTL(col interface{}) *runtime.Value {
	if col == nil {
		return runtime.Null
	}
	switch v := col.(type) {
	case []byte:
		return runtime.StringVal(string(v))
	case string:
		return runtime.StringVal(v)
	case int64:
		return runtime.NumberVal(float64(v))
	case float64:
		return runtime.NumberVal(v)
	case bool:
		return runtime.BoolVal(v)
	default:
		return runtime.StringVal(fmt.Sprintf("%v", v))
	}
}

func mysqlQuery(db *sql.DB, sqlStr string, params []interface{}) ([]*runtime.Value, error) {
	rows, err := db.Query(sqlStr, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var result []*runtime.Value
	for rows.Next() {
		scanVals := make([]interface{}, len(cols))
		scanPtrs := make([]interface{}, len(cols))
		for i := range scanVals {
			scanPtrs[i] = &scanVals[i]
		}
		if err := rows.Scan(scanPtrs...); err != nil {
			return nil, err
		}
		obj := make(map[string]*runtime.Value, len(cols))
		for i, col := range cols {
			obj[col] = mysqlColToNTL(scanVals[i])
		}
		result = append(result, runtime.ObjectVal(obj))
	}
	return result, rows.Err()
}

func mysqlExtractParams(args []*runtime.Value, idx int) []interface{} {
	var params []interface{}
	if len(args) > idx && args[idx].Tag == runtime.TypeArray {
		for _, p := range args[idx].ArrVal {
			if p == nil || p.Tag == runtime.TypeNull || p.Tag == runtime.TypeUndefined {
				params = append(params, nil)
			} else if p.Tag == runtime.TypeNumber {
				params = append(params, p.NumVal)
			} else if p.Tag == runtime.TypeBool {
				params = append(params, p.BoolVal)
			} else {
				params = append(params, p.ToString())
			}
		}
	}
	return params
}

func mysqlConnObj(db *sql.DB) *runtime.Value {
	query := runtime.FuncVal(&runtime.Function{
		Name: "query",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("query(sql, params?)")
			}
			rows, err := mysqlQuery(db, args[0].ToString(), mysqlExtractParams(args, 1))
			if err != nil {
				return runtime.Null, err
			}
			if rows == nil {
				rows = []*runtime.Value{}
			}
			return runtime.ArrayVal(rows), nil
		},
	})

	queryOne := runtime.FuncVal(&runtime.Function{
		Name: "queryOne",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("queryOne(sql, params?)")
			}
			rows, err := mysqlQuery(db, args[0].ToString(), mysqlExtractParams(args, 1))
			if err != nil || len(rows) == 0 {
				return runtime.Null, err
			}
			return rows[0], nil
		},
	})

	exec := runtime.FuncVal(&runtime.Function{
		Name: "exec",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("exec(sql, params?)")
			}
			result, err := db.Exec(args[0].ToString(), mysqlExtractParams(args, 1)...)
			if err != nil {
				return runtime.Null, err
			}
			affected, _ := result.RowsAffected()
			lastID, _ := result.LastInsertId()
			return runtime.ObjectVal(map[string]*runtime.Value{
				"rowsAffected": runtime.NumberVal(float64(affected)),
				"insertId":     runtime.NumberVal(float64(lastID)),
			}), nil
		},
	})

	transaction := runtime.FuncVal(&runtime.Function{
		Name: "transaction",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeFunction {
				return runtime.Null, fmt.Errorf("transaction(fn)")
			}
			tx, err := db.Begin()
			if err != nil {
				return runtime.Null, err
			}
			txQuery := runtime.FuncVal(&runtime.Function{
				Name: "query",
				Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					if len(a) == 0 {
						return runtime.Null, fmt.Errorf("query(sql)")
					}
					rows, err := tx.Query(a[0].ToString(), mysqlExtractParams(a, 1)...)
					if err != nil {
						return runtime.Null, err
					}
					defer rows.Close()
					cols, _ := rows.Columns()
					var result []*runtime.Value
					for rows.Next() {
						scanVals := make([]interface{}, len(cols))
						scanPtrs := make([]interface{}, len(cols))
						for i := range scanVals {
							scanPtrs[i] = &scanVals[i]
						}
						rows.Scan(scanPtrs...)
						obj := make(map[string]*runtime.Value, len(cols))
						for i, col := range cols {
							obj[col] = mysqlColToNTL(scanVals[i])
						}
						result = append(result, runtime.ObjectVal(obj))
					}
					return runtime.ArrayVal(result), nil
				},
			})
			txExec := runtime.FuncVal(&runtime.Function{
				Name: "exec",
				Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					if len(a) == 0 {
						return runtime.Null, fmt.Errorf("exec(sql)")
					}
					res, err := tx.Exec(a[0].ToString(), mysqlExtractParams(a, 1)...)
					if err != nil {
						return runtime.Null, err
					}
					affected, _ := res.RowsAffected()
					insertID, _ := res.LastInsertId()
					return runtime.ObjectVal(map[string]*runtime.Value{
						"rowsAffected": runtime.NumberVal(float64(affected)),
						"insertId":     runtime.NumberVal(float64(insertID)),
					}), nil
				},
			})
			txObj := runtime.ObjectVal(map[string]*runtime.Value{
				"query": txQuery,
				"exec":  txExec,
			})
			res, err := runtime.CallFunction(args[0], []*runtime.Value{txObj}, nil)
			if err != nil {
				tx.Rollback()
				return runtime.Null, err
			}
			if err := tx.Commit(); err != nil {
				return runtime.Null, err
			}
			return res, nil
		},
	})

	ping := runtime.FuncVal(&runtime.Function{
		Name: "ping",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.BoolVal(db.Ping() == nil), nil
		},
	})

	closeConn := runtime.FuncVal(&runtime.Function{
		Name: "close",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.Undefined, db.Close()
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"query":       query,
		"queryOne":    queryOne,
		"exec":        exec,
		"transaction": transaction,
		"ping":        ping,
		"close":       closeConn,
	})
}

func MySQLModule() *runtime.Value {
	connect := runtime.FuncVal(&runtime.Function{
		Name: "connect",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("connect(dsn) or connect({host, user, password, database})")
			}
			var dsn string
			if args[0].Tag == runtime.TypeString {
				dsn = args[0].ToString()
			} else if args[0].Tag == runtime.TypeObject {
				opts := args[0].ObjVal
				user := "root"
				password := ""
				host := "localhost"
				port := "3306"
				database := ""
				if v, ok := opts["user"]; ok {
					user = v.ToString()
				}
				if v, ok := opts["password"]; ok {
					password = v.ToString()
				}
				if v, ok := opts["host"]; ok {
					host = v.ToString()
				}
				if v, ok := opts["port"]; ok {
					port = v.ToString()
				}
				if v, ok := opts["database"]; ok {
					database = v.ToString()
				}
				dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, database)
			} else {
				return runtime.Null, fmt.Errorf("invalid connection options")
			}
			if existing, ok := mysqlPools.Load(dsn); ok {
				return mysqlConnObj(existing.(*sql.DB)), nil
			}
			db, err := sql.Open("mysql", dsn)
			if err != nil {
				return runtime.Null, err
			}
			if err := db.Ping(); err != nil {
				return runtime.Null, fmt.Errorf("cannot connect to MySQL: %w", err)
			}
			mysqlPools.Store(dsn, db)
			return mysqlConnObj(db), nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"connect": connect,
	})
}
