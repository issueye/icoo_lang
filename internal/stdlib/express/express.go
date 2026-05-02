package express

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

func LoadStdExpressModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.express",
		Path: "std.express",
		Exports: map[string]runtime.Value{
			"create":   &runtime.NativeFunction{Name: "create", Arity: 0, Fn: expressCreate},
			"json":     &runtime.NativeFunction{Name: "json", Arity: 1, Fn: expressJSON},
			"new":      &runtime.NativeFunction{Name: "new", Arity: 0, Fn: expressCreate},
			"next":     &runtime.NativeFunction{Name: "next", Arity: 0, Fn: expressNext},
			"redirect": &runtime.NativeFunction{Name: "redirect", Arity: -1, Fn: expressRedirect},
			"text":     &runtime.NativeFunction{Name: "text", Arity: -1, Fn: expressText},
		},
		Done: true,
	}
}

type appBinding struct {
	mu     sync.RWMutex
	routes []routeBinding
}

type routeBinding struct {
	method  string
	path    string
	handler runtime.Value
}

type serverBinding struct {
	server *http.Server
}

func expressCreate(args []runtime.Value) (runtime.Value, error) {
	app := &appBinding{}
	return app.object(), nil
}

func (app *appBinding) object() *runtime.ObjectValue {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"all":     app.routeFunction("all", "ALL"),
		"delete":  app.routeFunction("delete", http.MethodDelete),
		"get":     app.routeFunction("get", http.MethodGet),
		"head":    app.routeFunction("head", http.MethodHead),
		"listen":  &runtime.NativeFunction{Name: "express.listen", Arity: 1, CtxFn: app.listen},
		"options": app.routeFunction("options", http.MethodOptions),
		"patch":   app.routeFunction("patch", http.MethodPatch),
		"post":    app.routeFunction("post", http.MethodPost),
		"put":     app.routeFunction("put", http.MethodPut),
		"use":     app.routeFunction("use", "ALL"),
	}}
}

func (app *appBinding) routeFunction(name, method string) *runtime.NativeFunction {
	return &runtime.NativeFunction{
		Name:  "express." + name,
		Arity: -1,
		Fn: func(args []runtime.Value) (runtime.Value, error) {
			return app.addRoute(name, method, args)
		},
	}
}

func (app *appBinding) addRoute(name, method string, args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("%s expects handler or path and handler", name)
	}

	path := "/"
	handler := args[0]
	if len(args) == 2 {
		pathValue, err := requireStringArg(name, args[0])
		if err != nil {
			return nil, err
		}
		path = cleanRoutePath(pathValue)
		handler = args[1]
	}
	if !isCallableValue(handler) {
		return nil, fmt.Errorf("%s handler must be callable", name)
	}

	app.mu.Lock()
	app.routes = append(app.routes, routeBinding{method: method, path: path, handler: handler})
	app.mu.Unlock()
	return app.object(), nil
}

func (app *appBinding) listen(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
	addr, err := parseListenAddr(args[0])
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	binding := &serverBinding{}
	binding.server = &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.serveHTTP(ctx, w, r)
	})}

	go func() {
		_ = binding.server.Serve(listener)
	}()

	actualAddr := listener.Addr().String()
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"addr":  runtime.StringValue{Value: actualAddr},
		"close": &runtime.NativeFunction{Name: "express.server.close", Arity: 0, Fn: binding.close},
		"url":   runtime.StringValue{Value: "http://" + actualAddr},
	}}, nil
}

func (app *appBinding) serveHTTP(ctx *runtime.NativeContext, w http.ResponseWriter, r *http.Request) {
	reqValue, err := httpRequestToRuntime(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	app.mu.RLock()
	routes := append([]routeBinding(nil), app.routes...)
	app.mu.RUnlock()

	for _, route := range routes {
		if !route.matches(r.Method, r.URL.Path) {
			continue
		}
		respValue, nextReqValue, err := callRouteHandler(ctx, route.handler, reqValue)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if isNextResponse(respValue) {
			reqValue = nextReqValue
			continue
		}
		if err := writeResponse(w, respValue); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	http.NotFound(w, r)
}

func (route routeBinding) matches(method, path string) bool {
	if route.method != "ALL" && route.method != method {
		return false
	}
	if route.path == "*" {
		return true
	}
	if path == route.path {
		return true
	}
	if route.path == "/" {
		return route.method == "ALL"
	}
	return strings.HasPrefix(path, strings.TrimRight(route.path, "/")+"/")
}

func (binding *serverBinding) close(args []runtime.Value) (runtime.Value, error) {
	if binding == nil || binding.server == nil {
		return runtime.NullValue{}, nil
	}
	return runtime.NullValue{}, binding.server.Close()
}

func callRouteHandler(ctx *runtime.NativeContext, handler runtime.Value, reqValue runtime.Value) (runtime.Value, runtime.Value, error) {
	args := []runtime.Value{reqValue}
	if closure, ok := handler.(*runtime.Closure); ok && closure.Proto != nil && closure.Proto.Arity == 2 {
		args = []runtime.Value{reqValue, nextFunction()}
	} else if native, ok := handler.(*runtime.NativeFunction); ok && native.Arity == 2 {
		args = []runtime.Value{reqValue, nextFunction()}
	}
	if ctx.CallDetachedWithArgs != nil {
		result, calledArgs, err := ctx.CallDetachedWithArgs(handler, args)
		if len(calledArgs) > 0 {
			return result, calledArgs[0], err
		}
		return result, reqValue, err
	}
	result, err := ctx.CallDetached(handler, args)
	return result, reqValue, err
}

func nextFunction() *runtime.NativeFunction {
	return &runtime.NativeFunction{Name: "express.next", Arity: 0, Fn: expressNext}
}

func expressNext(args []runtime.Value) (runtime.Value, error) {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"next": runtime.BoolValue{Value: true},
	}}, nil
}

func expressJSON(args []runtime.Value) (runtime.Value, error) {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"json": args[0],
	}}, nil
}

func expressText(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("text expects body or status and body")
	}
	status := int64(http.StatusOK)
	bodyValue := args[0]
	if len(args) == 2 {
		intValue, ok := args[0].(runtime.IntValue)
		if !ok {
			return nil, fmt.Errorf("text status must be int")
		}
		status = intValue.Value
		bodyValue = args[1]
	}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"body":   runtime.StringValue{Value: bodyValue.String()},
		"status": runtime.IntValue{Value: status},
	}}, nil
}

func expressRedirect(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("redirect expects url or status and url")
	}
	status := int64(http.StatusFound)
	urlValue := args[0]
	if len(args) == 2 {
		intValue, ok := args[0].(runtime.IntValue)
		if !ok {
			return nil, fmt.Errorf("redirect status must be int")
		}
		status = intValue.Value
		urlValue = args[1]
	}
	url, err := requireStringArg("redirect", urlValue)
	if err != nil {
		return nil, err
	}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"headers": &runtime.ObjectValue{Fields: map[string]runtime.Value{
			"Location": runtime.StringValue{Value: url},
		}},
		"status": runtime.IntValue{Value: status},
	}}, nil
}

func parseListenAddr(value runtime.Value) (string, error) {
	switch v := value.(type) {
	case runtime.StringValue:
		if strings.TrimSpace(v.Value) == "" {
			return "", fmt.Errorf("listen addr must be non-empty")
		}
		return v.Value, nil
	case runtime.IntValue:
		if v.Value < 0 || v.Value > 65535 {
			return "", fmt.Errorf("listen port must be between 0 and 65535")
		}
		return fmt.Sprintf("127.0.0.1:%d", v.Value), nil
	case *runtime.ObjectValue:
		if addrValue, ok := v.Fields["addr"]; ok {
			addr, err := requireStringArg("listen", addrValue)
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(addr) == "" {
				return "", fmt.Errorf("listen addr must be non-empty")
			}
			return addr, nil
		}
		if portValue, ok := v.Fields["port"]; ok {
			port, ok := portValue.(runtime.IntValue)
			if !ok {
				return "", fmt.Errorf("listen port must be int")
			}
			if port.Value < 0 || port.Value > 65535 {
				return "", fmt.Errorf("listen port must be between 0 and 65535")
			}
			host := "127.0.0.1"
			if hostValue, ok := v.Fields["host"]; ok {
				var err error
				host, err = requireStringArg("listen", hostValue)
				if err != nil {
					return "", err
				}
			}
			return net.JoinHostPort(host, strconv.FormatInt(port.Value, 10)), nil
		}
		return "", fmt.Errorf("listen options require addr or port")
	default:
		return "", fmt.Errorf("listen expects string, int, or options object")
	}
}

func httpRequestToRuntime(r *http.Request) (runtime.Value, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
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

	req := &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"body":          runtime.StringValue{Value: string(body)},
		"contentLength": runtime.IntValue{Value: r.ContentLength},
		"header":        httpHeaderGetter(r.Header),
		"headers":       httpHeadersToRuntime(r.Header),
		"hasHeader":     httpHasHeaderGetter(r.Header),
		"host":          runtime.StringValue{Value: r.Host},
		"method":        runtime.StringValue{Value: r.Method},
		"path":          runtime.StringValue{Value: r.URL.Path},
		"query":         &runtime.ObjectValue{Fields: queryFields},
		"remoteAddr":    runtime.StringValue{Value: r.RemoteAddr},
		"url":           runtime.StringValue{Value: r.URL.String()},
	}}
	if len(body) > 0 && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var decoded any
		if err := json.Unmarshal(body, &decoded); err != nil {
			return nil, err
		}
		req.Fields["json"] = utils.PlainToRuntimeValue(decoded)
	}
	return req, nil
}

func writeResponse(w http.ResponseWriter, value runtime.Value) error {
	switch resp := value.(type) {
	case nil, runtime.NullValue:
		w.WriteHeader(http.StatusNoContent)
		return nil
	case runtime.StringValue:
		_, err := io.WriteString(w, resp.Value)
		return err
	case *runtime.ErrorValue:
		http.Error(w, resp.Message, http.StatusInternalServerError)
		return nil
	case *runtime.ObjectValue:
		status := http.StatusOK
		if statusValue, ok := resp.Fields["status"]; ok {
			intValue, ok := statusValue.(runtime.IntValue)
			if !ok {
				return fmt.Errorf("response status must be int")
			}
			status = int(intValue.Value)
		}

		if headersValue, ok := resp.Fields["headers"]; ok {
			headersObj, ok := headersValue.(*runtime.ObjectValue)
			if !ok {
				return fmt.Errorf("response headers must be object")
			}
			for key, headerValue := range headersObj.Fields {
				switch items := headerValue.(type) {
				case *runtime.ArrayValue:
					for _, item := range items.Elements {
						w.Header().Add(key, item.String())
					}
				default:
					w.Header().Set(key, headerValue.String())
				}
			}
		}

		if jsonValue, ok := resp.Fields["json"]; ok {
			plain, err := utils.RuntimeToPlainValue(jsonValue)
			if err != nil {
				return err
			}
			data, err := json.Marshal(plain)
			if err != nil {
				return err
			}
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", "application/json")
			}
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.WriteHeader(status)
			_, err = w.Write(data)
			return err
		}

		body := ""
		if bodyValue, ok := resp.Fields["body"]; ok {
			body = bodyValue.String()
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(status)
		_, err := io.WriteString(w, body)
		return err
	default:
		return fmt.Errorf("unsupported response value: %s", runtime.KindName(value))
	}
}

func httpHeadersToRuntime(headers http.Header) runtime.Value {
	fields := make(map[string]runtime.Value, len(headers))
	for key, values := range headers {
		if len(values) == 1 {
			fields[key] = runtime.StringValue{Value: values[0]}
			continue
		}
		items := make([]runtime.Value, 0, len(values))
		for _, value := range values {
			items = append(items, runtime.StringValue{Value: value})
		}
		fields[key] = &runtime.ArrayValue{Elements: items}
	}
	return &runtime.ObjectValue{Fields: fields}
}

func httpHeaderGetter(headers http.Header) runtime.Value {
	return &runtime.NativeFunction{
		Name:  "express.request.header",
		Arity: 1,
		Fn: func(args []runtime.Value) (runtime.Value, error) {
			name, err := requireStringArg("header", args[0])
			if err != nil {
				return nil, err
			}
			value := headers.Get(name)
			if value == "" {
				return runtime.NullValue{}, nil
			}
			return runtime.StringValue{Value: value}, nil
		},
	}
}

func httpHasHeaderGetter(headers http.Header) runtime.Value {
	return &runtime.NativeFunction{
		Name:  "express.request.hasHeader",
		Arity: 1,
		Fn: func(args []runtime.Value) (runtime.Value, error) {
			name, err := requireStringArg("hasHeader", args[0])
			if err != nil {
				return nil, err
			}
			return runtime.BoolValue{Value: headers.Get(name) != ""}, nil
		},
	}
}

func isNextResponse(value runtime.Value) bool {
	obj, ok := value.(*runtime.ObjectValue)
	if !ok {
		return false
	}
	next, ok := obj.Fields["next"].(runtime.BoolValue)
	return ok && next.Value
}

func cleanRoutePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if path == "*" {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func isCallableValue(value runtime.Value) bool {
	switch value.(type) {
	case *runtime.Closure, *runtime.NativeFunction:
		return true
	default:
		return false
	}
}

func requireStringArg(name string, v runtime.Value) (string, error) {
	text, ok := v.(runtime.StringValue)
	if !ok {
		return "", fmt.Errorf("%s expects string argument", name)
	}
	return text.Value, nil
}
