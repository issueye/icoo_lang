package stdnet

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

type webSocketConnBinding struct {
	conn *websocket.Conn
}

func LoadStdNetWebSocketClientModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.net.websocket.client",
		Path: "std.net.websocket.client",
		Exports: map[string]runtime.Value{
			"connect": &runtime.NativeFunction{Name: "connect", Arity: 1, Fn: webSocketConnect},
		},
		Done: true,
	}
}

func LoadStdNetWebSocketServerModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.net.websocket.server",
		Path: "std.net.websocket.server",
		Exports: map[string]runtime.Value{
			"listen": &runtime.NativeFunction{Name: "listen", Arity: 1, CtxFn: webSocketListen},
		},
		Done: true,
	}
}

func webSocketConnect(args []runtime.Value) (runtime.Value, error) {
	opts, err := parseNetURLTimeoutOptions("connect", args[0])
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, opts.URL, nil)
	if err != nil {
		return nil, err
	}
	return newWebSocketConnHandle(opts.URL, conn), nil
}

func webSocketListen(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
	obj, ok := args[0].(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("listen expects options object")
	}
	addrValue, ok := obj.Fields["addr"].(runtime.StringValue)
	if !ok || strings.TrimSpace(addrValue.Value) == "" {
		return nil, fmt.Errorf("listen options require non-empty addr")
	}
	path := "/"
	if pathValue, ok := obj.Fields["path"].(runtime.StringValue); ok && pathValue.Value != "" {
		path = pathValue.Value
	}
	handlerValue, ok := obj.Fields["handler"]
	if !ok {
		return nil, fmt.Errorf("listen options require handler")
	}
	if !isCallableValue(handlerValue) {
		return nil, fmt.Errorf("listen handler must be callable")
	}

	listener, err := net.Listen("tcp", addrValue.Value)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		reqValue, err := webSocketRequestToRuntime(r)
		if err != nil {
			_ = conn.Close(websocket.StatusInternalError, err.Error())
			return
		}
		_, err = ctx.CallDetached(handlerValue, []runtime.Value{newWebSocketConnHandle("", conn), reqValue})
		if err != nil {
			_ = conn.Close(websocket.StatusInternalError, err.Error())
		}
	})
	binding := &httpServerBinding{server: &http.Server{Handler: mux}}
	go func() {
		_ = binding.server.Serve(listener)
	}()
	addr := listener.Addr().String()
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"addr":  runtime.StringValue{Value: addr},
		"url":   runtime.StringValue{Value: "ws://" + addr + path},
		"close": &runtime.NativeFunction{Name: "close", Arity: 0, Fn: binding.close},
	}}, nil
}

func newWebSocketConnHandle(url string, conn *websocket.Conn) runtime.Value {
	binding := &webSocketConnBinding{conn: conn}
	fields := map[string]runtime.Value{
		"url":        runtime.StringValue{Value: url},
		"localAddr":  runtime.StringValue{Value: ""},
		"remoteAddr": runtime.StringValue{Value: ""},
		"read":       &runtime.NativeFunction{Name: "read", Arity: 0, Fn: binding.read},
		"write":      &runtime.NativeFunction{Name: "write", Arity: 1, Fn: binding.write},
		"close":      &runtime.NativeFunction{Name: "close", Arity: 0, Fn: binding.close},
	}
	return &runtime.ObjectValue{Fields: fields}
}

func (binding *webSocketConnBinding) read(args []runtime.Value) (runtime.Value, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	msgType, data, err := binding.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	if msgType != websocket.MessageText {
		return nil, fmt.Errorf("read expects text message")
	}
	return runtime.StringValue{Value: string(data)}, nil
}

func (binding *webSocketConnBinding) write(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("write", args[0])
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := binding.conn.Write(ctx, websocket.MessageText, []byte(text)); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

func (binding *webSocketConnBinding) close(args []runtime.Value) (runtime.Value, error) {
	if binding == nil || binding.conn == nil {
		return runtime.NullValue{}, nil
	}
	return runtime.NullValue{}, binding.conn.Close(websocket.StatusNormalClosure, "")
}

func webSocketRequestToRuntime(r *http.Request) (runtime.Value, error) {
	queryFields := make(map[string]runtime.Value, len(r.URL.Query()))
	for key, values := range r.URL.Query() {
		if len(values) == 1 {
			queryFields[key] = runtime.StringValue{Value: values[0]}
			continue
		}
		items := make([]runtime.Value, 0, len(values))
		for _, value := range values {
			items = append(items, runtime.StringValue{Value: value})
		}
		queryFields[key] = &runtime.ArrayValue{Elements: items}
	}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"method":     runtime.StringValue{Value: r.Method},
		"url":        runtime.StringValue{Value: r.URL.String()},
		"path":       runtime.StringValue{Value: r.URL.Path},
		"query":      &runtime.ObjectValue{Fields: queryFields},
		"headers":    httpHeadersToRuntime(r.Header),
		"host":       runtime.StringValue{Value: r.Host},
		"remoteAddr": runtime.StringValue{Value: r.RemoteAddr},
	}}, nil
}

type netURLTimeoutOptions struct {
	URL     string
	Timeout time.Duration
}

func parseNetURLTimeoutOptions(name string, v runtime.Value) (*netURLTimeoutOptions, error) {
	obj, ok := v.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s expects options object", name)
	}
	urlValue, ok := obj.Fields["url"].(runtime.StringValue)
	if !ok || strings.TrimSpace(urlValue.Value) == "" {
		return nil, fmt.Errorf("%s options require non-empty url", name)
	}
	timeout, err := parseOptionalTimeout(obj, name, 30*time.Second)
	if err != nil {
		return nil, err
	}
	return &netURLTimeoutOptions{URL: urlValue.Value, Timeout: timeout}, nil
}
