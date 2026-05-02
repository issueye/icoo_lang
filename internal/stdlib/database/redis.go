package database

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"

	redis "github.com/redis/go-redis/v9"
)

type redisHandle struct {
	client *redis.Client
}

type redisHandleValue struct {
	handle *redisHandle
}

func (v *redisHandleValue) Kind() runtime.ValueKind { return runtime.ObjectKind }

func (v *redisHandleValue) String() string {
	if v == nil || v.handle == nil || v.handle.client == nil {
		return "<redis closed>"
	}
	return "<redis>"
}

func LoadStdRedisModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.redis",
		Path: "std.redis",
		Exports: map[string]runtime.Value{
			"connect": &runtime.NativeFunction{Name: "connect", Arity: 1, Fn: redisConnect},
			"open":    &runtime.NativeFunction{Name: "open", Arity: 1, Fn: redisOpen},
		},
		Done: true,
	}
}

func redisOpen(args []runtime.Value) (runtime.Value, error) {
	url, err := utils.RequireStringArg("open", args[0])
	if err != nil {
		return nil, err
	}
	options, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return newRedisObject(redis.NewClient(options)), nil
}

func redisConnect(args []runtime.Value) (runtime.Value, error) {
	options, err := redisOptionsFromRuntime(args[0])
	if err != nil {
		return nil, err
	}
	return newRedisObject(redis.NewClient(options)), nil
}

func newRedisObject(client *redis.Client) *runtime.ObjectValue {
	handle := &redisHandle{client: client}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"close": &runtime.NativeFunction{Name: "redis.close", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			if handle.client == nil {
				return runtime.NullValue{}, nil
			}
			if err := handle.client.Close(); err != nil {
				return nil, err
			}
			handle.client = nil
			return runtime.NullValue{}, nil
		}},
		"del": &runtime.NativeFunction{Name: "redis.del", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("del")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("del", args[0])
			if err != nil {
				return nil, err
			}
			count, err := client.Del(context.Background(), key).Result()
			if err != nil {
				return nil, err
			}
			return runtime.IntValue{Value: count}, nil
		}},
		"exists": &runtime.NativeFunction{Name: "redis.exists", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("exists")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("exists", args[0])
			if err != nil {
				return nil, err
			}
			count, err := client.Exists(context.Background(), key).Result()
			if err != nil {
				return nil, err
			}
			return runtime.BoolValue{Value: count > 0}, nil
		}},
		"expire": &runtime.NativeFunction{Name: "redis.expire", Arity: 2, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("expire")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("expire", args[0])
			if err != nil {
				return nil, err
			}
			ttl, err := redisDurationArg("expire", args[1])
			if err != nil {
				return nil, err
			}
			ok, err := client.PExpire(context.Background(), key, ttl).Result()
			if err != nil {
				return nil, err
			}
			return runtime.BoolValue{Value: ok}, nil
		}},
		"get": &runtime.NativeFunction{Name: "redis.get", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("get")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("get", args[0])
			if err != nil {
				return nil, err
			}
			value, err := client.Get(context.Background(), key).Result()
			if err == redis.Nil {
				return runtime.NullValue{}, nil
			}
			if err != nil {
				return nil, err
			}
			return runtime.StringValue{Value: value}, nil
		}},
		"hGet": &runtime.NativeFunction{Name: "redis.hGet", Arity: 2, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("hGet")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("hGet", args[0])
			if err != nil {
				return nil, err
			}
			field, err := utils.RequireStringArg("hGet", args[1])
			if err != nil {
				return nil, err
			}
			value, err := client.HGet(context.Background(), key, field).Result()
			if err == redis.Nil {
				return runtime.NullValue{}, nil
			}
			if err != nil {
				return nil, err
			}
			return runtime.StringValue{Value: value}, nil
		}},
		"hGetAll": &runtime.NativeFunction{Name: "redis.hGetAll", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("hGetAll")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("hGetAll", args[0])
			if err != nil {
				return nil, err
			}
			values, err := client.HGetAll(context.Background(), key).Result()
			if err != nil {
				return nil, err
			}
			fields := make(map[string]runtime.Value, len(values))
			for field, value := range values {
				fields[field] = runtime.StringValue{Value: value}
			}
			return &runtime.ObjectValue{Fields: fields}, nil
		}},
		"hSet": &runtime.NativeFunction{Name: "redis.hSet", Arity: 2, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("hSet")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("hSet", args[0])
			if err != nil {
				return nil, err
			}
			data, ok := args[1].(*runtime.ObjectValue)
			if !ok {
				return nil, fmt.Errorf("hSet expects object argument")
			}
			keys := make([]string, 0, len(data.Fields))
			for field := range data.Fields {
				keys = append(keys, field)
			}
			sort.Strings(keys)
			values := make([]any, 0, len(keys)*2)
			for _, field := range keys {
				value, err := redisValueString("hSet", data.Fields[field])
				if err != nil {
					return nil, err
				}
				values = append(values, field, value)
			}
			count, err := client.HSet(context.Background(), key, values...).Result()
			if err != nil {
				return nil, err
			}
			return runtime.IntValue{Value: count}, nil
		}},
		"incr": &runtime.NativeFunction{Name: "redis.incr", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("incr")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("incr", args[0])
			if err != nil {
				return nil, err
			}
			value, err := client.Incr(context.Background(), key).Result()
			if err != nil {
				return nil, err
			}
			return runtime.IntValue{Value: value}, nil
		}},
		"incrBy": &runtime.NativeFunction{Name: "redis.incrBy", Arity: 2, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("incrBy")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("incrBy", args[0])
			if err != nil {
				return nil, err
			}
			delta, err := redisIntArg("incrBy", args[1])
			if err != nil {
				return nil, err
			}
			value, err := client.IncrBy(context.Background(), key, delta).Result()
			if err != nil {
				return nil, err
			}
			return runtime.IntValue{Value: value}, nil
		}},
		"ping": &runtime.NativeFunction{Name: "redis.ping", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("ping")
			if err != nil {
				return nil, err
			}
			if err := client.Ping(context.Background()).Err(); err != nil {
				return nil, err
			}
			return runtime.BoolValue{Value: true}, nil
		}},
		"set": &runtime.NativeFunction{Name: "redis.set", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("set")
			if err != nil {
				return nil, err
			}
			if len(args) < 2 || len(args) > 3 {
				return nil, fmt.Errorf("set expects 2 or 3 arguments")
			}
			key, err := utils.RequireStringArg("set", args[0])
			if err != nil {
				return nil, err
			}
			value, err := redisValueString("set", args[1])
			if err != nil {
				return nil, err
			}
			ttl := time.Duration(0)
			if len(args) == 3 {
				ttl, err = redisDurationArg("set", args[2])
				if err != nil {
					return nil, err
				}
			}
			result, err := client.Set(context.Background(), key, value, ttl).Result()
			if err != nil {
				return nil, err
			}
			return runtime.BoolValue{Value: strings.EqualFold(result, "OK")}, nil
		}},
		"ttl": &runtime.NativeFunction{Name: "redis.ttl", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("ttl")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("ttl", args[0])
			if err != nil {
				return nil, err
			}
			value, err := client.PTTL(context.Background(), key).Result()
			if err != nil {
				return nil, err
			}
			if value < 0 {
				return runtime.NullValue{}, nil
			}
			return runtime.IntValue{Value: value.Milliseconds()}, nil
		}},
	}}
}

func (h *redisHandle) requireClient(name string) (*redis.Client, error) {
	if h == nil || h.client == nil {
		return nil, fmt.Errorf("%s called on closed redis client", name)
	}
	return h.client, nil
}

func redisOptionsFromRuntime(value runtime.Value) (*redis.Options, error) {
	obj, ok := value.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("connect expects options object")
	}

	if urlValue, ok := obj.Fields["url"]; ok {
		url, err := utils.RequireStringArg("connect", urlValue)
		if err != nil {
			return nil, err
		}
		options, err := redis.ParseURL(url)
		if err != nil {
			return nil, err
		}
		if err := applyRedisOptionOverrides(options, obj); err != nil {
			return nil, err
		}
		return options, nil
	}

	addrValue, ok := obj.Fields["addr"]
	if !ok {
		return nil, fmt.Errorf("connect options require addr or url")
	}
	addr, err := utils.RequireStringArg("connect", addrValue)
	if err != nil {
		return nil, err
	}
	options := &redis.Options{Addr: strings.TrimSpace(addr)}
	if options.Addr == "" {
		return nil, fmt.Errorf("connect options require non-empty addr")
	}
	if err := applyRedisOptionOverrides(options, obj); err != nil {
		return nil, err
	}
	return options, nil
}

func applyRedisOptionOverrides(options *redis.Options, obj *runtime.ObjectValue) error {
	if usernameValue, ok := obj.Fields["username"]; ok {
		username, err := utils.RequireStringArg("connect", usernameValue)
		if err != nil {
			return err
		}
		options.Username = username
	}
	if passwordValue, ok := obj.Fields["password"]; ok {
		password, err := utils.RequireStringArg("connect", passwordValue)
		if err != nil {
			return err
		}
		options.Password = password
	}
	if dbValue, ok := obj.Fields["db"]; ok {
		db, err := redisIntArg("connect", dbValue)
		if err != nil {
			return err
		}
		if db < 0 {
			return fmt.Errorf("connect db must be non-negative")
		}
		options.DB = int(db)
	}
	if poolSizeValue, ok := obj.Fields["poolSize"]; ok {
		poolSize, err := redisIntArg("connect", poolSizeValue)
		if err != nil {
			return err
		}
		if poolSize < 0 {
			return fmt.Errorf("connect poolSize must be non-negative")
		}
		options.PoolSize = int(poolSize)
	}
	if dialTimeoutValue, ok := obj.Fields["dialTimeoutMs"]; ok {
		timeout, err := redisDurationArg("connect", dialTimeoutValue)
		if err != nil {
			return err
		}
		options.DialTimeout = timeout
	}
	if readTimeoutValue, ok := obj.Fields["readTimeoutMs"]; ok {
		timeout, err := redisDurationArg("connect", readTimeoutValue)
		if err != nil {
			return err
		}
		options.ReadTimeout = timeout
	}
	if writeTimeoutValue, ok := obj.Fields["writeTimeoutMs"]; ok {
		timeout, err := redisDurationArg("connect", writeTimeoutValue)
		if err != nil {
			return err
		}
		options.WriteTimeout = timeout
	}
	return nil
}

func redisIntArg(name string, value runtime.Value) (int64, error) {
	intValue, ok := value.(runtime.IntValue)
	if !ok {
		return 0, fmt.Errorf("%s expects int argument", name)
	}
	return intValue.Value, nil
}

func redisDurationArg(name string, value runtime.Value) (time.Duration, error) {
	ms, err := redisIntArg(name, value)
	if err != nil {
		return 0, err
	}
	if ms < 0 {
		return 0, fmt.Errorf("%s expects non-negative milliseconds", name)
	}
	return time.Duration(ms) * time.Millisecond, nil
}

func redisValueString(name string, value runtime.Value) (string, error) {
	switch typed := value.(type) {
	case runtime.StringValue:
		return typed.Value, nil
	case runtime.IntValue, runtime.FloatValue, runtime.BoolValue:
		return value.String(), nil
	case runtime.NullValue:
		return "", nil
	case *runtime.ArrayValue, *runtime.ObjectValue:
		plain, err := utils.RuntimeToPlainValue(value)
		if err != nil {
			return "", err
		}
		data, err := json.Marshal(plain)
		if err != nil {
			return "", fmt.Errorf("%s encode value: %w", name, err)
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("%s does not support %s value", name, runtime.KindName(value))
	}
}
