// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"fmt"
	"lunex/internal/runtime"

	"github.com/graphql-go/graphql"
)

func gqlValueToNTL(v interface{}) *runtime.Value {
	return jsonToValue(v)
}

func GraphQLModule() *runtime.Value {
	buildSchema := runtime.FuncVal(&runtime.Function{
		Name: "buildSchema",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("buildSchema({types, query, mutation?})")
			}
			opts := args[0].ObjVal

			fields := graphql.Fields{}

			if queryObj, ok := opts["query"]; ok && queryObj.Tag == runtime.TypeObject {
				for fieldName, fieldVal := range queryObj.ObjVal {
					fn := fieldName
					fv := fieldVal
					if fv.Tag != runtime.TypeFunction {
						continue
					}
					fields[fn] = &graphql.Field{
						Type: graphql.String,
						Args: graphql.FieldConfigArgument{
							"input": &graphql.ArgumentConfig{Type: graphql.String},
						},
						Resolve: func(p graphql.ResolveParams) (interface{}, error) {
							argVal := runtime.ObjectVal(map[string]*runtime.Value{})
							for k, v := range p.Args {
								argVal.ObjVal[k] = jsonToValue(v)
							}
							result, err := runtime.CallFunction(fv, []*runtime.Value{argVal}, nil)
							if err != nil {
								return nil, err
							}
							if result == nil {
								return nil, nil
							}
							switch result.Tag {
							case runtime.TypeString:
								return result.StrVal, nil
							case runtime.TypeNumber:
								return result.NumVal, nil
							case runtime.TypeBool:
								return result.BoolVal, nil
							case runtime.TypeNull, runtime.TypeUndefined:
								return nil, nil
							default:
								return result.Inspect(), nil
							}
						},
					}
				}
			}

			queryType := graphql.NewObject(graphql.ObjectConfig{
				Name:   "Query",
				Fields: fields,
			})

			schema, err := graphql.NewSchema(graphql.SchemaConfig{
				Query: queryType,
			})
			if err != nil {
				return runtime.Null, err
			}

			execute := runtime.FuncVal(&runtime.Function{
				Name: "execute",
				Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					if len(a) == 0 {
						return runtime.Null, fmt.Errorf("execute(query, variables?)")
					}
					query := a[0].ToString()
					vars := map[string]interface{}{}
					if len(a) > 1 && a[1].Tag == runtime.TypeObject {
						for k, v := range a[1].ObjVal {
							switch v.Tag {
							case runtime.TypeString:
								vars[k] = v.StrVal
							case runtime.TypeNumber:
								vars[k] = v.NumVal
							case runtime.TypeBool:
								vars[k] = v.BoolVal
							}
						}
					}
					result := graphql.Do(graphql.Params{
						Schema:         schema,
						RequestString:  query,
						VariableValues: vars,
					})
					if len(result.Errors) > 0 {
						errMsgs := make([]*runtime.Value, len(result.Errors))
						for i, e := range result.Errors {
							errMsgs[i] = runtime.StringVal(e.Error())
						}
						return runtime.ObjectVal(map[string]*runtime.Value{
							"errors": runtime.ArrayVal(errMsgs),
							"data":   runtime.Null,
						}), nil
					}
					data := gqlValueToNTL(result.Data)
					return runtime.ObjectVal(map[string]*runtime.Value{
						"data":   data,
						"errors": runtime.ArrayVal(nil),
					}), nil
				},
			})

			executeRaw := runtime.FuncVal(&runtime.Function{
				Name: "executeRaw",
				Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
					if len(a) == 0 {
						return runtime.Null, fmt.Errorf("executeRaw(query)")
					}
					result := graphql.Do(graphql.Params{
						Schema:        schema,
						RequestString: a[0].ToString(),
					})
					data := gqlValueToNTL(result.Data)
					return data, nil
				},
			})

			return runtime.ObjectVal(map[string]*runtime.Value{
				"execute":    execute,
				"executeRaw": executeRaw,
			}), nil
		},
	})

	execute := runtime.FuncVal(&runtime.Function{
		Name: "execute",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("execute(schema, query, variables?)")
			}
			return runtime.Null, fmt.Errorf("use buildSchema(...).execute(query) instead")
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"buildSchema": buildSchema,
		"execute":     execute,
	})
}
