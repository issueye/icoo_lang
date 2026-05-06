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

type redisPipelineBinding struct {
	handle   *redisHandle
	commands []redisPipelineCommand
}

type redisPipelineCommand struct {
	run func(pipe redis.Pipeliner) redis.Cmder
}

type redisSubscriptionBinding struct {
	pubsub *redis.PubSub
}

func (v *redisHandleValue) Kind() runtime.ValueKind { return runtime.ObjectKind }

func (v *redisHandleValue) String() string {
	if v == nil || v.handle == nil || v.handle.client == nil {
		return "<redis closed>"
	}
	return "<redis>"
}

func LoadStdDBRedisModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.db.redis",
		Path: "std.db.redis",
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
		"lPop": &runtime.NativeFunction{Name: "redis.lPop", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("lPop")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("lPop", args[0])
			if err != nil {
				return nil, err
			}
			value, err := client.LPop(context.Background(), key).Result()
			if err == redis.Nil {
				return runtime.NullValue{}, nil
			}
			if err != nil {
				return nil, err
			}
			return runtime.StringValue{Value: value}, nil
		}},
		"lPush": &runtime.NativeFunction{Name: "redis.lPush", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("lPush")
			if err != nil {
				return nil, err
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("lPush expects key and at least one value")
			}
			key, err := utils.RequireStringArg("lPush", args[0])
			if err != nil {
				return nil, err
			}
			values, err := redisArgsToAny("lPush", args[1:])
			if err != nil {
				return nil, err
			}
			count, err := client.LPush(context.Background(), key, values...).Result()
			if err != nil {
				return nil, err
			}
			return runtime.IntValue{Value: count}, nil
		}},
		"lRange": &runtime.NativeFunction{Name: "redis.lRange", Arity: 3, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("lRange")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("lRange", args[0])
			if err != nil {
				return nil, err
			}
			start, err := redisIntArg("lRange", args[1])
			if err != nil {
				return nil, err
			}
			stop, err := redisIntArg("lRange", args[2])
			if err != nil {
				return nil, err
			}
			values, err := client.LRange(context.Background(), key, start, stop).Result()
			if err != nil {
				return nil, err
			}
			return redisStringsToRuntimeArray(values), nil
		}},
		"pipeline": &runtime.NativeFunction{Name: "redis.pipeline", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			if _, err := handle.requireClient("pipeline"); err != nil {
				return nil, err
			}
			return newRedisPipelineObject(handle), nil
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
		"publish": &runtime.NativeFunction{Name: "redis.publish", Arity: 2, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("publish")
			if err != nil {
				return nil, err
			}
			channel, err := utils.RequireStringArg("publish", args[0])
			if err != nil {
				return nil, err
			}
			message, err := redisValueString("publish", args[1])
			if err != nil {
				return nil, err
			}
			count, err := client.Publish(context.Background(), channel, message).Result()
			if err != nil {
				return nil, err
			}
			return runtime.IntValue{Value: count}, nil
		}},
		"rPop": &runtime.NativeFunction{Name: "redis.rPop", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("rPop")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("rPop", args[0])
			if err != nil {
				return nil, err
			}
			value, err := client.RPop(context.Background(), key).Result()
			if err == redis.Nil {
				return runtime.NullValue{}, nil
			}
			if err != nil {
				return nil, err
			}
			return runtime.StringValue{Value: value}, nil
		}},
		"rPush": &runtime.NativeFunction{Name: "redis.rPush", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("rPush")
			if err != nil {
				return nil, err
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("rPush expects key and at least one value")
			}
			key, err := utils.RequireStringArg("rPush", args[0])
			if err != nil {
				return nil, err
			}
			values, err := redisArgsToAny("rPush", args[1:])
			if err != nil {
				return nil, err
			}
			count, err := client.RPush(context.Background(), key, values...).Result()
			if err != nil {
				return nil, err
			}
			return runtime.IntValue{Value: count}, nil
		}},
		"sAdd": &runtime.NativeFunction{Name: "redis.sAdd", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("sAdd")
			if err != nil {
				return nil, err
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("sAdd expects key and at least one value")
			}
			key, err := utils.RequireStringArg("sAdd", args[0])
			if err != nil {
				return nil, err
			}
			values, err := redisArgsToAny("sAdd", args[1:])
			if err != nil {
				return nil, err
			}
			count, err := client.SAdd(context.Background(), key, values...).Result()
			if err != nil {
				return nil, err
			}
			return runtime.IntValue{Value: count}, nil
		}},
		"sIsMember": &runtime.NativeFunction{Name: "redis.sIsMember", Arity: 2, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("sIsMember")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("sIsMember", args[0])
			if err != nil {
				return nil, err
			}
			value, err := redisValueString("sIsMember", args[1])
			if err != nil {
				return nil, err
			}
			ok, err := client.SIsMember(context.Background(), key, value).Result()
			if err != nil {
				return nil, err
			}
			return runtime.BoolValue{Value: ok}, nil
		}},
		"sMembers": &runtime.NativeFunction{Name: "redis.sMembers", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("sMembers")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("sMembers", args[0])
			if err != nil {
				return nil, err
			}
			values, err := client.SMembers(context.Background(), key).Result()
			if err != nil {
				return nil, err
			}
			sort.Strings(values)
			return redisStringsToRuntimeArray(values), nil
		}},
		"sRem": &runtime.NativeFunction{Name: "redis.sRem", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("sRem")
			if err != nil {
				return nil, err
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("sRem expects key and at least one value")
			}
			key, err := utils.RequireStringArg("sRem", args[0])
			if err != nil {
				return nil, err
			}
			values, err := redisArgsToAny("sRem", args[1:])
			if err != nil {
				return nil, err
			}
			count, err := client.SRem(context.Background(), key, values...).Result()
			if err != nil {
				return nil, err
			}
			return runtime.IntValue{Value: count}, nil
		}},
		"scan": &runtime.NativeFunction{Name: "redis.scan", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("scan")
			if err != nil {
				return nil, err
			}
			if len(args) < 1 || len(args) > 3 {
				return nil, fmt.Errorf("scan expects cursor and optional pattern/count")
			}
			cursor, err := redisIntArg("scan", args[0])
			if err != nil {
				return nil, err
			}
			pattern := "*"
			if len(args) >= 2 {
				pattern, err = utils.RequireStringArg("scan", args[1])
				if err != nil {
					return nil, err
				}
			}
			count := int64(10)
			if len(args) == 3 {
				count, err = redisIntArg("scan", args[2])
				if err != nil {
					return nil, err
				}
			}
			keys, nextCursor, err := client.Scan(context.Background(), uint64(cursor), pattern, count).Result()
			if err != nil {
				return nil, err
			}
			return &runtime.ObjectValue{Fields: map[string]runtime.Value{
				"cursor": runtime.IntValue{Value: int64(nextCursor)},
				"keys":   redisStringsToRuntimeArray(keys),
			}}, nil
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
		"subscribe": &runtime.NativeFunction{Name: "redis.subscribe", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("subscribe")
			if err != nil {
				return nil, err
			}
			if len(args) == 0 {
				return nil, fmt.Errorf("subscribe expects at least one channel")
			}
			channels := make([]string, 0, len(args))
			for _, arg := range args {
				channel, err := utils.RequireStringArg("subscribe", arg)
				if err != nil {
					return nil, err
				}
				channels = append(channels, channel)
			}
			pubsub := client.Subscribe(context.Background(), channels...)
			if _, err := pubsub.Receive(context.Background()); err != nil {
				_ = pubsub.Close()
				return nil, err
			}
			return newRedisSubscriptionObject(pubsub), nil
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
		"zAdd": &runtime.NativeFunction{Name: "redis.zAdd", Arity: 3, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("zAdd")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("zAdd", args[0])
			if err != nil {
				return nil, err
			}
			member, err := redisValueString("zAdd", args[1])
			if err != nil {
				return nil, err
			}
			score, err := redisFloatArg("zAdd", args[2])
			if err != nil {
				return nil, err
			}
			count, err := client.ZAdd(context.Background(), key, redis.Z{Member: member, Score: score}).Result()
			if err != nil {
				return nil, err
			}
			return runtime.IntValue{Value: count}, nil
		}},
		"zRange": &runtime.NativeFunction{Name: "redis.zRange", Arity: 3, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("zRange")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("zRange", args[0])
			if err != nil {
				return nil, err
			}
			start, err := redisIntArg("zRange", args[1])
			if err != nil {
				return nil, err
			}
			stop, err := redisIntArg("zRange", args[2])
			if err != nil {
				return nil, err
			}
			values, err := client.ZRange(context.Background(), key, start, stop).Result()
			if err != nil {
				return nil, err
			}
			return redisStringsToRuntimeArray(values), nil
		}},
		"zRem": &runtime.NativeFunction{Name: "redis.zRem", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("zRem")
			if err != nil {
				return nil, err
			}
			if len(args) < 2 {
				return nil, fmt.Errorf("zRem expects key and at least one member")
			}
			key, err := utils.RequireStringArg("zRem", args[0])
			if err != nil {
				return nil, err
			}
			values, err := redisArgsToAny("zRem", args[1:])
			if err != nil {
				return nil, err
			}
			count, err := client.ZRem(context.Background(), key, values...).Result()
			if err != nil {
				return nil, err
			}
			return runtime.IntValue{Value: count}, nil
		}},
		"zScore": &runtime.NativeFunction{Name: "redis.zScore", Arity: 2, Fn: func(args []runtime.Value) (runtime.Value, error) {
			client, err := handle.requireClient("zScore")
			if err != nil {
				return nil, err
			}
			key, err := utils.RequireStringArg("zScore", args[0])
			if err != nil {
				return nil, err
			}
			member, err := redisValueString("zScore", args[1])
			if err != nil {
				return nil, err
			}
			score, err := client.ZScore(context.Background(), key, member).Result()
			if err == redis.Nil {
				return runtime.NullValue{}, nil
			}
			if err != nil {
				return nil, err
			}
			return utils.PlainToRuntimeValue(score), nil
		}},
	}}
}

func newRedisPipelineObject(handle *redisHandle) *runtime.ObjectValue {
	return redisPipelineBindingObject(&redisPipelineBinding{handle: handle})
}

func redisPipelineBindingObject(binding *redisPipelineBinding) *runtime.ObjectValue {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"del": &runtime.NativeFunction{Name: "redis.pipeline.del", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			key, err := utils.RequireStringArg("pipeline.del", args[0])
			if err != nil {
				return nil, err
			}
			binding.commands = append(binding.commands, redisPipelineCommand{
				run: func(pipe redis.Pipeliner) redis.Cmder {
					return pipe.Del(context.Background(), key)
				},
			})
			return redisPipelineBindingObject(binding), nil
		}},
		"exec": &runtime.NativeFunction{Name: "redis.pipeline.exec", Arity: 0, Fn: binding.exec},
		"get": &runtime.NativeFunction{Name: "redis.pipeline.get", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			key, err := utils.RequireStringArg("pipeline.get", args[0])
			if err != nil {
				return nil, err
			}
			binding.commands = append(binding.commands, redisPipelineCommand{
				run: func(pipe redis.Pipeliner) redis.Cmder {
					return pipe.Get(context.Background(), key)
				},
			})
			return redisPipelineBindingObject(binding), nil
		}},
		"incr": &runtime.NativeFunction{Name: "redis.pipeline.incr", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			key, err := utils.RequireStringArg("pipeline.incr", args[0])
			if err != nil {
				return nil, err
			}
			binding.commands = append(binding.commands, redisPipelineCommand{
				run: func(pipe redis.Pipeliner) redis.Cmder {
					return pipe.Incr(context.Background(), key)
				},
			})
			return redisPipelineBindingObject(binding), nil
		}},
		"set": &runtime.NativeFunction{Name: "redis.pipeline.set", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			if len(args) < 2 || len(args) > 3 {
				return nil, fmt.Errorf("pipeline.set expects key, value, and optional ttl")
			}
			key, err := utils.RequireStringArg("pipeline.set", args[0])
			if err != nil {
				return nil, err
			}
			value, err := redisValueString("pipeline.set", args[1])
			if err != nil {
				return nil, err
			}
			ttl := time.Duration(0)
			if len(args) == 3 {
				ttl, err = redisDurationArg("pipeline.set", args[2])
				if err != nil {
					return nil, err
				}
			}
			binding.commands = append(binding.commands, redisPipelineCommand{
				run: func(pipe redis.Pipeliner) redis.Cmder {
					return pipe.Set(context.Background(), key, value, ttl)
				},
			})
			return redisPipelineBindingObject(binding), nil
		}},
	}}
}

func (binding *redisPipelineBinding) exec(args []runtime.Value) (runtime.Value, error) {
	client, err := binding.handle.requireClient("pipeline.exec")
	if err != nil {
		return nil, err
	}
	pipe := client.Pipeline()
	cmds := make([]redis.Cmder, 0, len(binding.commands))
	for _, command := range binding.commands {
		cmds = append(cmds, command.run(pipe))
	}
	_, err = pipe.Exec(context.Background())
	if err != nil && err != redis.Nil {
		return nil, err
	}
	results := make([]runtime.Value, 0, len(cmds))
	for _, cmd := range cmds {
		value, err := redisCmdToRuntime(cmd)
		if err != nil {
			return nil, err
		}
		results = append(results, value)
	}
	binding.commands = nil
	return &runtime.ArrayValue{Elements: results}, nil
}

func newRedisSubscriptionObject(pubsub *redis.PubSub) *runtime.ObjectValue {
	binding := &redisSubscriptionBinding{pubsub: pubsub}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"close":   &runtime.NativeFunction{Name: "redis.subscription.close", Arity: 0, Fn: binding.close},
		"receive": &runtime.NativeFunction{Name: "redis.subscription.receive", Arity: -1, Fn: binding.receive},
	}}
}

func (binding *redisSubscriptionBinding) receive(args []runtime.Value) (runtime.Value, error) {
	timeout := 30 * time.Second
	if len(args) > 1 {
		return nil, fmt.Errorf("receive expects optional timeout")
	}
	if len(args) == 1 {
		ms, err := redisDurationArg("receive", args[0])
		if err != nil {
			return nil, err
		}
		timeout = ms
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	message, err := binding.pubsub.ReceiveMessage(ctx)
	if err != nil {
		if err == context.DeadlineExceeded {
			return runtime.NullValue{}, nil
		}
		return nil, err
	}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"channel": runtime.StringValue{Value: message.Channel},
		"payload": runtime.StringValue{Value: message.Payload},
	}}, nil
}

func (binding *redisSubscriptionBinding) close(args []runtime.Value) (runtime.Value, error) {
	if binding.pubsub == nil {
		return runtime.NullValue{}, nil
	}
	return runtime.NullValue{}, binding.pubsub.Close()
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

func redisFloatArg(name string, value runtime.Value) (float64, error) {
	switch typed := value.(type) {
	case runtime.IntValue:
		return float64(typed.Value), nil
	case runtime.FloatValue:
		return typed.Value, nil
	default:
		return 0, fmt.Errorf("%s expects numeric argument", name)
	}
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

func redisArgsToAny(name string, args []runtime.Value) ([]any, error) {
	values := make([]any, 0, len(args))
	for _, arg := range args {
		value, err := redisValueString(name, arg)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func redisStringsToRuntimeArray(values []string) runtime.Value {
	items := make([]runtime.Value, 0, len(values))
	for _, value := range values {
		items = append(items, runtime.StringValue{Value: value})
	}
	return &runtime.ArrayValue{Elements: items}
}

func redisCmdToRuntime(cmd redis.Cmder) (runtime.Value, error) {
	switch typed := cmd.(type) {
	case *redis.StatusCmd:
		value, err := typed.Result()
		if err != nil {
			return nil, err
		}
		return runtime.StringValue{Value: value}, nil
	case *redis.StringCmd:
		value, err := typed.Result()
		if err == redis.Nil {
			return runtime.NullValue{}, nil
		}
		if err != nil {
			return nil, err
		}
		return runtime.StringValue{Value: value}, nil
	case *redis.IntCmd:
		value, err := typed.Result()
		if err != nil {
			return nil, err
		}
		return runtime.IntValue{Value: value}, nil
	default:
		return runtime.StringValue{Value: cmd.String()}, nil
	}
}
