// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"context"
	"fmt"
	"lunex/internal/runtime"
	"time"

	"github.com/redis/go-redis/v9"
)

func redisClientObj(rdb *redis.Client) *runtime.Value {
	ctx := context.Background()

	set := runtime.FuncVal(&runtime.Function{
		Name: "set",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("set(key, value, ttlMs?)")
			}
			key := args[0].ToString()
			val := args[1].ToString()
			var ttl time.Duration
			if len(args) > 2 && args[2].Tag == runtime.TypeNumber {
				ttl = time.Duration(args[2].NumVal) * time.Millisecond
			}
			if err := rdb.Set(ctx, key, val, ttl).Err(); err != nil {
				return runtime.Null, err
			}
			return runtime.True, nil
		},
	})

	get := runtime.FuncVal(&runtime.Function{
		Name: "get",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("get(key)")
			}
			val, err := rdb.Get(ctx, args[0].ToString()).Result()
			if err == redis.Nil {
				return runtime.Null, nil
			}
			if err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(val), nil
		},
	})

	del := runtime.FuncVal(&runtime.Function{
		Name: "del",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(0), nil
			}
			keys := make([]string, len(args))
			for i, a := range args {
				keys[i] = a.ToString()
			}
			n, err := rdb.Del(ctx, keys...).Result()
			if err != nil {
				return runtime.Null, err
			}
			return runtime.NumberVal(float64(n)), nil
		},
	})

	exists := runtime.FuncVal(&runtime.Function{
		Name: "exists",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			n, err := rdb.Exists(ctx, args[0].ToString()).Result()
			if err != nil {
				return runtime.False, err
			}
			return runtime.BoolVal(n > 0), nil
		},
	})

	expire := runtime.FuncVal(&runtime.Function{
		Name: "expire",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, fmt.Errorf("expire(key, ttlMs)")
			}
			ttl := time.Duration(args[1].NumVal) * time.Millisecond
			ok, err := rdb.Expire(ctx, args[0].ToString(), ttl).Result()
			if err != nil {
				return runtime.False, err
			}
			return runtime.BoolVal(ok), nil
		},
	})

	ttlFn := runtime.FuncVal(&runtime.Function{
		Name: "ttl",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(-1), nil
			}
			d, err := rdb.TTL(ctx, args[0].ToString()).Result()
			if err != nil {
				return runtime.NumberVal(-1), err
			}
			return runtime.NumberVal(float64(d.Milliseconds())), nil
		},
	})

	keys := runtime.FuncVal(&runtime.Function{
		Name: "keys",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			pattern := "*"
			if len(args) > 0 {
				pattern = args[0].ToString()
			}
			ks, err := rdb.Keys(ctx, pattern).Result()
			if err != nil {
				return runtime.ArrayVal(nil), err
			}
			arr := make([]*runtime.Value, len(ks))
			for i, k := range ks {
				arr[i] = runtime.StringVal(k)
			}
			return runtime.ArrayVal(arr), nil
		},
	})

	incr := runtime.FuncVal(&runtime.Function{
		Name: "incr",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("incr(key)")
			}
			n, err := rdb.Incr(ctx, args[0].ToString()).Result()
			if err != nil {
				return runtime.Null, err
			}
			return runtime.NumberVal(float64(n)), nil
		},
	})

	incrBy := runtime.FuncVal(&runtime.Function{
		Name: "incrBy",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("incrBy(key, amount)")
			}
			n, err := rdb.IncrBy(ctx, args[0].ToString(), int64(args[1].NumVal)).Result()
			if err != nil {
				return runtime.Null, err
			}
			return runtime.NumberVal(float64(n)), nil
		},
	})

	decr := runtime.FuncVal(&runtime.Function{
		Name: "decr",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("decr(key)")
			}
			n, err := rdb.Decr(ctx, args[0].ToString()).Result()
			if err != nil {
				return runtime.Null, err
			}
			return runtime.NumberVal(float64(n)), nil
		},
	})

	hset := runtime.FuncVal(&runtime.Function{
		Name: "hset",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				return runtime.Null, fmt.Errorf("hset(key, field, value)")
			}
			err := rdb.HSet(ctx, args[0].ToString(), args[1].ToString(), args[2].ToString()).Err()
			if err != nil {
				return runtime.False, err
			}
			return runtime.True, nil
		},
	})

	hget := runtime.FuncVal(&runtime.Function{
		Name: "hget",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("hget(key, field)")
			}
			val, err := rdb.HGet(ctx, args[0].ToString(), args[1].ToString()).Result()
			if err == redis.Nil {
				return runtime.Null, nil
			}
			if err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(val), nil
		},
	})

	hgetall := runtime.FuncVal(&runtime.Function{
		Name: "hgetall",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ObjectVal(nil), nil
			}
			m, err := rdb.HGetAll(ctx, args[0].ToString()).Result()
			if err != nil {
				return runtime.ObjectVal(nil), err
			}
			obj := make(map[string]*runtime.Value, len(m))
			for k, v := range m {
				obj[k] = runtime.StringVal(v)
			}
			return runtime.ObjectVal(obj), nil
		},
	})

	hdel := runtime.FuncVal(&runtime.Function{
		Name: "hdel",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.NumberVal(0), nil
			}
			n, err := rdb.HDel(ctx, args[0].ToString(), args[1].ToString()).Result()
			if err != nil {
				return runtime.NumberVal(0), err
			}
			return runtime.NumberVal(float64(n)), nil
		},
	})

	lpush := runtime.FuncVal(&runtime.Function{
		Name: "lpush",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("lpush(key, value)")
			}
			n, err := rdb.LPush(ctx, args[0].ToString(), args[1].ToString()).Result()
			if err != nil {
				return runtime.Null, err
			}
			return runtime.NumberVal(float64(n)), nil
		},
	})

	rpush := runtime.FuncVal(&runtime.Function{
		Name: "rpush",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("rpush(key, value)")
			}
			n, err := rdb.RPush(ctx, args[0].ToString(), args[1].ToString()).Result()
			if err != nil {
				return runtime.Null, err
			}
			return runtime.NumberVal(float64(n)), nil
		},
	})

	lpop := runtime.FuncVal(&runtime.Function{
		Name: "lpop",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("lpop(key)")
			}
			val, err := rdb.LPop(ctx, args[0].ToString()).Result()
			if err == redis.Nil {
				return runtime.Null, nil
			}
			if err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(val), nil
		},
	})

	rpop := runtime.FuncVal(&runtime.Function{
		Name: "rpop",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("rpop(key)")
			}
			val, err := rdb.RPop(ctx, args[0].ToString()).Result()
			if err == redis.Nil {
				return runtime.Null, nil
			}
			if err != nil {
				return runtime.Null, err
			}
			return runtime.StringVal(val), nil
		},
	})

	lrange := runtime.FuncVal(&runtime.Function{
		Name: "lrange",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				return runtime.ArrayVal(nil), fmt.Errorf("lrange(key, start, stop)")
			}
			vals, err := rdb.LRange(ctx, args[0].ToString(), int64(args[1].NumVal), int64(args[2].NumVal)).Result()
			if err != nil {
				return runtime.ArrayVal(nil), err
			}
			arr := make([]*runtime.Value, len(vals))
			for i, v := range vals {
				arr[i] = runtime.StringVal(v)
			}
			return runtime.ArrayVal(arr), nil
		},
	})

	sadd := runtime.FuncVal(&runtime.Function{
		Name: "sadd",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("sadd(key, member)")
			}
			n, err := rdb.SAdd(ctx, args[0].ToString(), args[1].ToString()).Result()
			if err != nil {
				return runtime.Null, err
			}
			return runtime.NumberVal(float64(n)), nil
		},
	})

	smembers := runtime.FuncVal(&runtime.Function{
		Name: "smembers",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), nil
			}
			members, err := rdb.SMembers(ctx, args[0].ToString()).Result()
			if err != nil {
				return runtime.ArrayVal(nil), err
			}
			arr := make([]*runtime.Value, len(members))
			for i, m := range members {
				arr[i] = runtime.StringVal(m)
			}
			return runtime.ArrayVal(arr), nil
		},
	})

	sismember := runtime.FuncVal(&runtime.Function{
		Name: "sismember",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, nil
			}
			ok, err := rdb.SIsMember(ctx, args[0].ToString(), args[1].ToString()).Result()
			if err != nil {
				return runtime.False, err
			}
			return runtime.BoolVal(ok), nil
		},
	})

	publish := runtime.FuncVal(&runtime.Function{
		Name: "publish",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("publish(channel, message)")
			}
			n, err := rdb.Publish(ctx, args[0].ToString(), args[1].ToString()).Result()
			if err != nil {
				return runtime.Null, err
			}
			return runtime.NumberVal(float64(n)), nil
		},
	})

	subscribe := runtime.FuncVal(&runtime.Function{
		Name: "subscribe",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[1].Tag != runtime.TypeFunction {
				return runtime.Null, fmt.Errorf("subscribe(channel, handler)")
			}
			channel := args[0].ToString()
			handler := args[1]
			sub := rdb.Subscribe(ctx, channel)
			runtime.KeepAliveAdd()
			go func() {
				defer runtime.KeepAliveDone()
				ch := sub.Channel()
				for msg := range ch {
					if runtime.CallFunction != nil {
						runtime.CallFunction(handler, []*runtime.Value{
							runtime.StringVal(msg.Payload),
							runtime.StringVal(msg.Channel),
						}, nil)
					}
				}
			}()
			return runtime.ObjectVal(map[string]*runtime.Value{
				"unsubscribe": runtime.FuncVal(&runtime.Function{
					Name: "unsubscribe",
					Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
						sub.Close()
						return runtime.Undefined, nil
					},
				}),
			}), nil
		},
	})

	flushdb := runtime.FuncVal(&runtime.Function{
		Name: "flushdb",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.BoolVal(rdb.FlushDB(ctx).Err() == nil), nil
		},
	})

	ping := runtime.FuncVal(&runtime.Function{
		Name: "ping",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.BoolVal(rdb.Ping(ctx).Err() == nil), nil
		},
	})

	closeConn := runtime.FuncVal(&runtime.Function{
		Name: "close",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.Undefined, rdb.Close()
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"set":       set,
		"get":       get,
		"del":       del,
		"exists":    exists,
		"expire":    expire,
		"ttl":       ttlFn,
		"keys":      keys,
		"incr":      incr,
		"incrBy":    incrBy,
		"decr":      decr,
		"hset":      hset,
		"hget":      hget,
		"hgetall":   hgetall,
		"hdel":      hdel,
		"lpush":     lpush,
		"rpush":     rpush,
		"lpop":      lpop,
		"rpop":      rpop,
		"lrange":    lrange,
		"sadd":      sadd,
		"smembers":  smembers,
		"sismember": sismember,
		"publish":   publish,
		"subscribe": subscribe,
		"flushdb":   flushdb,
		"ping":      ping,
		"close":     closeConn,
	})
}

func RedisModule() *runtime.Value {
	connect := runtime.FuncVal(&runtime.Function{
		Name: "connect",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			addr := "localhost:6379"
			password := ""
			db := 0
			if len(args) > 0 && args[0].Tag == runtime.TypeObject {
				opts := args[0].ObjVal
				if v, ok := opts["addr"]; ok {
					addr = v.ToString()
				}
				if v, ok := opts["host"]; ok {
					port := "6379"
					if p, ok2 := opts["port"]; ok2 {
						port = p.ToString()
					}
					addr = v.ToString() + ":" + port
				}
				if v, ok := opts["password"]; ok {
					password = v.ToString()
				}
				if v, ok := opts["url"]; ok {
					opt, err := redis.ParseURL(v.ToString())
					if err != nil {
						return runtime.Null, err
					}
					rdb := redis.NewClient(opt)
					return redisClientObj(rdb), nil
				}
				if v, ok := opts["db"]; ok && v.Tag == runtime.TypeNumber {
					db = int(v.NumVal)
				}
			} else if len(args) > 0 && args[0].Tag == runtime.TypeString {
				opt, err := redis.ParseURL(args[0].ToString())
				if err != nil {
					addr = args[0].ToString()
				} else {
					rdb := redis.NewClient(opt)
					return redisClientObj(rdb), nil
				}
			}
			rdb := redis.NewClient(&redis.Options{
				Addr:     addr,
				Password: password,
				DB:       db,
			})
			return redisClientObj(rdb), nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"connect": connect,
	})
}
