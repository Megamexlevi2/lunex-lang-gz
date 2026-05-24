// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"context"
	"fmt"
	"lunex/internal/runtime"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pgPools sync.Map

func pgRowToValue(rows pgx.Rows) ([]*runtime.Value, error) {
	fields := rows.FieldDescriptions()
	var result []*runtime.Value
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		obj := make(map[string]*runtime.Value, len(fields))
		for i, f := range fields {
			obj[string(f.Name)] = pgValToNTL(vals[i])
		}
		result = append(result, runtime.ObjectVal(obj))
	}
	return result, rows.Err()
}

func pgValToNTL(v interface{}) *runtime.Value {
	if v == nil {
		return runtime.Null
	}
	switch val := v.(type) {
	case bool:
		return runtime.BoolVal(val)
	case int16:
		return runtime.NumberVal(float64(val))
	case int32:
		return runtime.NumberVal(float64(val))
	case int64:
		return runtime.NumberVal(float64(val))
	case float32:
		return runtime.NumberVal(float64(val))
	case float64:
		return runtime.NumberVal(val)
	case string:
		return runtime.StringVal(val)
	case []byte:
		return runtime.StringVal(string(val))
	default:
		return runtime.StringVal(fmt.Sprintf("%v", v))
	}
}

func ntlValToPg(v *runtime.Value) interface{} {
	if v == nil || v.Tag == runtime.TypeNull || v.Tag == runtime.TypeUndefined {
		return nil
	}
	switch v.Tag {
	case runtime.TypeBool:
		return v.BoolVal
	case runtime.TypeNumber:
		return v.NumVal
	case runtime.TypeString:
		return v.StrVal
	default:
		return v.ToString()
	}
}

func pgConnObj(pool *pgxpool.Pool) *runtime.Value {
	ctx := context.Background()

	query := runtime.FuncVal(&runtime.Function{
		Name: "query",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("query requires a SQL string")
			}
			sql := args[0].ToString()
			var params []interface{}
			if len(args) > 1 && args[1].Tag == runtime.TypeArray {
				for _, p := range args[1].ArrVal {
					params = append(params, ntlValToPg(p))
				}
			}
			rows, err := pool.Query(ctx, sql, params...)
			if err != nil {
				return runtime.Null, err
			}
			defer rows.Close()
			result, err := pgRowToValue(rows)
			if err != nil {
				return runtime.Null, err
			}
			if result == nil {
				result = []*runtime.Value{}
			}
			return runtime.ArrayVal(result), nil
		},
	})

	queryOne := runtime.FuncVal(&runtime.Function{
		Name: "queryOne",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("queryOne requires a SQL string")
			}
			sql := args[0].ToString()
			var params []interface{}
			if len(args) > 1 && args[1].Tag == runtime.TypeArray {
				for _, p := range args[1].ArrVal {
					params = append(params, ntlValToPg(p))
				}
			}
			rows, err := pool.Query(ctx, sql, params...)
			if err != nil {
				return runtime.Null, err
			}
			defer rows.Close()
			result, err := pgRowToValue(rows)
			if err != nil || len(result) == 0 {
				return runtime.Null, err
			}
			return result[0], nil
		},
	})

	exec := runtime.FuncVal(&runtime.Function{
		Name: "exec",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("exec requires a SQL string")
			}
			sql := args[0].ToString()
			var params []interface{}
			if len(args) > 1 && args[1].Tag == runtime.TypeArray {
				for _, p := range args[1].ArrVal {
					params = append(params, ntlValToPg(p))
				}
			}
			tag, err := pool.Exec(ctx, sql, params...)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"rowsAffected": runtime.NumberVal(float64(tag.RowsAffected())),
				"command":      runtime.StringVal(tag.String()),
			}), nil
		},
	})

	transaction := runtime.FuncVal(&runtime.Function{
		Name: "transaction",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeFunction {
				return runtime.Null, fmt.Errorf("transaction requires a function")
			}
			fn := args[0]
			tx, err := pool.Begin(ctx)
			if err != nil {
				return runtime.Null, err
			}
			txQuery := runtime.FuncVal(&runtime.Function{
				Name: "query",
				Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					if len(a) == 0 {
						return runtime.Null, fmt.Errorf("query requires SQL")
					}
					sql := a[0].ToString()
					var params []interface{}
					if len(a) > 1 && a[1].Tag == runtime.TypeArray {
						for _, p := range a[1].ArrVal {
							params = append(params, ntlValToPg(p))
						}
					}
					rows, err := tx.Query(ctx, sql, params...)
					if err != nil {
						return runtime.Null, err
					}
					defer rows.Close()
					result, err := pgRowToValue(rows)
					if err != nil {
						return runtime.Null, err
					}
					if result == nil {
						result = []*runtime.Value{}
					}
					return runtime.ArrayVal(result), nil
				},
			})
			txExec := runtime.FuncVal(&runtime.Function{
				Name: "exec",
				Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					if len(a) == 0 {
						return runtime.Null, fmt.Errorf("exec requires SQL")
					}
					sql := a[0].ToString()
					var params []interface{}
					if len(a) > 1 && a[1].Tag == runtime.TypeArray {
						for _, p := range a[1].ArrVal {
							params = append(params, ntlValToPg(p))
						}
					}
					tag, err := tx.Exec(ctx, sql, params...)
					if err != nil {
						return runtime.Null, err
					}
					return runtime.ObjectVal(map[string]*runtime.Value{
						"rowsAffected": runtime.NumberVal(float64(tag.RowsAffected())),
					}), nil
				},
			})
			txObj := runtime.ObjectVal(map[string]*runtime.Value{
				"query": txQuery,
				"exec":  txExec,
			})
			result, err := runtime.CallFunction(fn, []*runtime.Value{txObj}, nil)
			if err != nil {
				_ = tx.Rollback(ctx)
				return runtime.Null, err
			}
			if err := tx.Commit(ctx); err != nil {
				return runtime.Null, err
			}
			return result, nil
		},
	})

	closeConn := runtime.FuncVal(&runtime.Function{
		Name: "close",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			pool.Close()
			return runtime.Undefined, nil
		},
	})

	ping := runtime.FuncVal(&runtime.Function{
		Name: "ping",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if err := pool.Ping(ctx); err != nil {
				return runtime.False, nil
			}
			return runtime.True, nil
		},
	})

	stats := runtime.FuncVal(&runtime.Function{
		Name: "stats",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			s := pool.Stat()
			return runtime.ObjectVal(map[string]*runtime.Value{
				"totalConns":    runtime.NumberVal(float64(s.TotalConns())),
				"idleConns":     runtime.NumberVal(float64(s.IdleConns())),
				"acquiredConns": runtime.NumberVal(float64(s.AcquiredConns())),
			}), nil
		},
	})

	insert := runtime.FuncVal(&runtime.Function{
		Name: "insert",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("insert(table, data) requires table name and data object")
			}
			table := args[0].ToString()
			if args[1].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("insert data must be an object")
			}
			var cols []string
			var placeholders []string
			var params []interface{}
			i := 1
			for k, v := range args[1].ObjVal {
				cols = append(cols, k)
				placeholders = append(placeholders, fmt.Sprintf("$%d", i))
				params = append(params, ntlValToPg(v))
				i++
			}
			sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING *",
				table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
			rows, err := pool.Query(ctx, sql, params...)
			if err != nil {
				return runtime.Null, err
			}
			defer rows.Close()
			result, err := pgRowToValue(rows)
			if err != nil || len(result) == 0 {
				return runtime.Null, err
			}
			return result[0], nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"query":       query,
		"queryOne":    queryOne,
		"exec":        exec,
		"transaction": transaction,
		"insert":      insert,
		"ping":        ping,
		"stats":       stats,
		"close":       closeConn,
	})
}

func PostgresModule() *runtime.Value {
	connect := runtime.FuncVal(&runtime.Function{
		Name: "connect",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("connect requires a connection string")
			}
			dsn := args[0].ToString()
			poolKey := dsn

			if existing, ok := pgPools.Load(poolKey); ok {
				return pgConnObj(existing.(*pgxpool.Pool)), nil
			}

			config, err := pgxpool.ParseConfig(dsn)
			if err != nil {
				return runtime.Null, fmt.Errorf("invalid connection string: %w", err)
			}

			pool, err := pgxpool.NewWithConfig(context.Background(), config)
			if err != nil {
				return runtime.Null, fmt.Errorf("failed to connect: %w", err)
			}

			pgPools.Store(poolKey, pool)
			return pgConnObj(pool), nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"connect": connect,
	})
}
