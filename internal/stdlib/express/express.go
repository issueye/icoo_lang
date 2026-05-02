package express

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

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
	method     string
	path       string
	handler    runtime.Value
	middleware bool
}

type serverBinding struct {
	server *http.Server
}

type responseBinding struct {
	writer      http.ResponseWriter
	flusher     http.Flusher
	statusCode  int
	wroteHeader bool
	handled     bool
	handle      *runtime.ObjectValue
}

type requestOptions struct {
	Method          string
	URL             string
	Headers         map[string]string
	Body            string
	Timeout         time.Duration
	Host            string
	Cookies         map[string]string
	FollowRedirects bool
	MaxRedirects    int
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
	app.routes = append(app.routes, routeBinding{
		method:     method,
		path:       path,
		handler:    handler,
		middleware: name == "use",
	})
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
	reqObject := reqValue.(*runtime.ObjectValue)
	requestID, _ := reqObject.Fields["requestId"].(runtime.StringValue)
	if requestID.Value != "" {
		w.Header().Set("X-Request-Id", requestID.Value)
	}

	app.mu.RLock()
	routes := append([]routeBinding(nil), app.routes...)
	app.mu.RUnlock()
	respBinding := newResponseHandle(w)

	for _, route := range routes {
		if !route.matches(r.Method, r.URL.Path) {
			continue
		}
		respValue, nextReqValue, err := callRouteHandler(ctx, route, reqValue, respBinding)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if respBinding.handled {
			return
		}
		if isNextResponse(respValue) {
			reqValue = nextReqValue
			continue
		}
		if err := writeResponse(w, respValue, respBinding.statusCode); err != nil {
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

func callRouteHandler(ctx *runtime.NativeContext, route routeBinding, reqValue runtime.Value, respBinding *responseBinding) (runtime.Value, runtime.Value, error) {
	handler := route.handler
	args := []runtime.Value{reqValue}
	if closure, ok := handler.(*runtime.Closure); ok && closure.Proto != nil && closure.Proto.Arity == 2 {
		if route.middleware {
			args = []runtime.Value{reqValue, nextFunction()}
		} else {
			args = []runtime.Value{reqValue, respBinding.handle}
		}
	} else if native, ok := handler.(*runtime.NativeFunction); ok && native.Arity == 2 {
		if route.middleware {
			args = []runtime.Value{reqValue, nextFunction()}
		} else {
			args = []runtime.Value{reqValue, respBinding.handle}
		}
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

func newResponseHandle(w http.ResponseWriter) *responseBinding {
	flusher, _ := w.(http.Flusher)
	binding := &responseBinding{
		writer:     w,
		flusher:    flusher,
		statusCode: http.StatusOK,
	}
	binding.handle = &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"proxy":       &runtime.NativeFunction{Name: "express.response.proxy", Arity: 2, Fn: binding.proxy},
		"statusCode":  &runtime.NativeFunction{Name: "express.response.statusCode", Arity: 0, Fn: binding.statusCodeValue},
		"status":      &runtime.NativeFunction{Name: "express.response.status", Arity: 1, Fn: binding.status},
		"setHeader":   &runtime.NativeFunction{Name: "express.response.setHeader", Arity: 2, Fn: binding.setHeader},
		"setCookie":   &runtime.NativeFunction{Name: "express.response.setCookie", Arity: -1, Fn: binding.setCookie},
		"clearCookie": &runtime.NativeFunction{Name: "express.response.clearCookie", Arity: -1, Fn: binding.clearCookie},
		"sse":         &runtime.NativeFunction{Name: "express.response.sse", Arity: 1, Fn: binding.writeSSE},
		"write":       &runtime.NativeFunction{Name: "express.response.write", Arity: 1, Fn: binding.write},
		"json":        &runtime.NativeFunction{Name: "express.response.json", Arity: 1, Fn: binding.writeJSON},
		"flush":       &runtime.NativeFunction{Name: "express.response.flush", Arity: 0, Fn: binding.flush},
		"end":         &runtime.NativeFunction{Name: "express.response.end", Arity: -1, Fn: binding.end},
	}}
	return binding
}

func (binding *responseBinding) statusCodeValue(args []runtime.Value) (runtime.Value, error) {
	return runtime.IntValue{Value: int64(binding.statusCode)}, nil
}

func (binding *responseBinding) status(args []runtime.Value) (runtime.Value, error) {
	code, ok := args[0].(runtime.IntValue)
	if !ok {
		return nil, fmt.Errorf("response.status expects int argument")
	}
	binding.statusCode = int(code.Value)
	return binding.handle, nil
}

func (binding *responseBinding) setHeader(args []runtime.Value) (runtime.Value, error) {
	name, err := requireStringArg("response.setHeader", args[0])
	if err != nil {
		return nil, err
	}
	binding.writer.Header().Set(name, args[1].String())
	return binding.handle, nil
}

func (binding *responseBinding) setCookie(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("response.setCookie expects name, value, and optional options")
	}
	name, err := requireStringArg("response.setCookie", args[0])
	if err != nil {
		return nil, err
	}
	value, err := requireStringArg("response.setCookie", args[1])
	if err != nil {
		return nil, err
	}
	cookie, err := httpCookieFromRuntime(name, value, args[2:])
	if err != nil {
		return nil, err
	}
	http.SetCookie(binding.writer, cookie)
	return binding.handle, nil
}

func (binding *responseBinding) clearCookie(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("response.clearCookie expects name and optional options")
	}
	name, err := requireStringArg("response.clearCookie", args[0])
	if err != nil {
		return nil, err
	}
	cookie, err := httpCookieFromRuntime(name, "", args[1:])
	if err != nil {
		return nil, err
	}
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(0, 0)
	http.SetCookie(binding.writer, cookie)
	return binding.handle, nil
}

func (binding *responseBinding) write(args []runtime.Value) (runtime.Value, error) {
	if err := binding.ensureHeader(); err != nil {
		return nil, err
	}
	binding.handled = true
	_, err := io.WriteString(binding.writer, args[0].String())
	if err != nil {
		return nil, err
	}
	return binding.handle, nil
}

func (binding *responseBinding) writeSSE(args []runtime.Value) (runtime.Value, error) {
	if binding.writer.Header().Get("Content-Type") == "" {
		binding.writer.Header().Set("Content-Type", "text/event-stream")
	}
	if err := binding.ensureHeader(); err != nil {
		return nil, err
	}
	binding.handled = true
	text, err := stdnetFormatSSEEvent(args[0])
	if err != nil {
		return nil, err
	}
	if _, err := io.WriteString(binding.writer, text); err != nil {
		return nil, err
	}
	if binding.flusher != nil {
		binding.flusher.Flush()
	}
	return binding.handle, nil
}

func (binding *responseBinding) writeJSON(args []runtime.Value) (runtime.Value, error) {
	plain, err := utils.RuntimeToPlainValue(args[0])
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(plain)
	if err != nil {
		return nil, err
	}
	if binding.writer.Header().Get("Content-Type") == "" {
		binding.writer.Header().Set("Content-Type", "application/json")
	}
	if err := binding.ensureHeader(); err != nil {
		return nil, err
	}
	binding.handled = true
	_, err = binding.writer.Write(data)
	if err != nil {
		return nil, err
	}
	return binding.handle, nil
}

func (binding *responseBinding) flush(args []runtime.Value) (runtime.Value, error) {
	if binding.flusher == nil {
		return runtime.NullValue{}, nil
	}
	if err := binding.ensureHeader(); err != nil {
		return nil, err
	}
	binding.handled = true
	binding.flusher.Flush()
	return binding.handle, nil
}

func (binding *responseBinding) end(args []runtime.Value) (runtime.Value, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("response.end expects 0 or 1 arguments")
	}
	if len(args) == 1 {
		return binding.write(args)
	}
	if err := binding.ensureHeader(); err != nil {
		return nil, err
	}
	binding.handled = true
	return binding.handle, nil
}

func (binding *responseBinding) proxy(args []runtime.Value) (runtime.Value, error) {
	opts, responseHeaders, stripHeaders, err := buildForwardRequestOptions("response.proxy", args[0], args[1])
	if err != nil {
		return nil, err
	}

	resp, err := doHTTPRoundTrip(opts)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		if isHopByHopHeader(key) {
			continue
		}
		lower := strings.ToLower(key)
		if _, ok := stripHeaders[lower]; ok {
			continue
		}
		binding.writer.Header().Del(key)
		for _, value := range values {
			binding.writer.Header().Add(key, value)
		}
	}
	for key, value := range responseHeaders {
		binding.writer.Header().Set(key, value)
	}

	binding.statusCode = resp.StatusCode
	if err := binding.ensureHeader(); err != nil {
		return nil, err
	}
	binding.handled = true

	target := io.Writer(binding.writer)
	if binding.flusher != nil {
		target = &flushingWriter{
			writer:  binding.writer,
			flusher: binding.flusher,
		}
	}
	if _, err := io.Copy(target, resp.Body); err != nil {
		return nil, err
	}
	return binding.handle, nil
}

func (binding *responseBinding) ensureHeader() error {
	if binding == nil || binding.writer == nil {
		return fmt.Errorf("response writer is unavailable")
	}
	if binding.wroteHeader {
		return nil
	}
	if binding.statusCode == 0 {
		binding.statusCode = http.StatusOK
	}
	binding.writer.WriteHeader(binding.statusCode)
	binding.wroteHeader = true
	return nil
}

func stdnetFormatSSEEvent(v runtime.Value) (string, error) {
	switch value := v.(type) {
	case runtime.StringValue:
		return "data: " + strings.ReplaceAll(value.Value, "\n", "\ndata: ") + "\n\n", nil
	case *runtime.ObjectValue:
		var b strings.Builder
		if event, ok := value.Fields["event"].(runtime.StringValue); ok && event.Value != "" {
			b.WriteString("event: ")
			b.WriteString(event.Value)
			b.WriteString("\n")
		}
		if id, ok := value.Fields["id"].(runtime.StringValue); ok {
			b.WriteString("id: ")
			b.WriteString(id.Value)
			b.WriteString("\n")
		}
		if retry, ok := value.Fields["retry"].(runtime.IntValue); ok {
			b.WriteString("retry: ")
			b.WriteString(strconv.FormatInt(retry.Value, 10))
			b.WriteString("\n")
		}
		data := ""
		if dataValue, ok := value.Fields["data"]; ok {
			data = dataValue.String()
		}
		b.WriteString("data: ")
		b.WriteString(strings.ReplaceAll(data, "\n", "\ndata: "))
		b.WriteString("\n\n")
		return b.String(), nil
	default:
		return "", fmt.Errorf("sse expects string or object")
	}
}

type flushingWriter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

func (w *flushingWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if err == nil && n > 0 && w.flusher != nil {
		w.flusher.Flush()
	}
	return n, err
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
	requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if requestID == "" {
		requestID = utils.GenerateRequestID()
	}
	queryFields := make(map[string]runtime.Value, len(r.URL.Query()))
	for key, values := range r.URL.Query() {
		queryFields[key] = httpStringValuesToRuntime(values)
	}

	formValue, filesValue, err := httpRequestBodyToRuntime(r.Header.Get("Content-Type"), body)
	if err != nil {
		return nil, err
	}

	req := &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"body":          runtime.StringValue{Value: string(body)},
		"cookie":        httpCookieGetter(r.Cookies()),
		"cookies":       httpCookiesToRuntime(r.Cookies()),
		"contentLength": runtime.IntValue{Value: r.ContentLength},
		"file":          httpFileGetter(filesValue),
		"files":         filesValue,
		"form":          formValue,
		"header":        httpHeaderGetter(r.Header),
		"headers":       httpHeadersToRuntime(r.Header),
		"hasHeader":     httpHasHeaderGetter(r.Header),
		"host":          runtime.StringValue{Value: r.Host},
		"method":        runtime.StringValue{Value: r.Method},
		"path":          runtime.StringValue{Value: r.URL.Path},
		"query":         &runtime.ObjectValue{Fields: queryFields},
		"remoteAddr":    runtime.StringValue{Value: r.RemoteAddr},
		"requestId":     runtime.StringValue{Value: requestID},
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

func writeResponse(w http.ResponseWriter, value runtime.Value, defaultStatus int) error {
	if defaultStatus == 0 {
		defaultStatus = http.StatusOK
	}
	switch resp := value.(type) {
	case nil, runtime.NullValue:
		status := defaultStatus
		if status == http.StatusOK {
			status = http.StatusNoContent
		}
		w.WriteHeader(status)
		return nil
	case runtime.StringValue:
		w.Header().Set("Content-Length", strconv.Itoa(len(resp.Value)))
		if defaultStatus != http.StatusOK {
			w.WriteHeader(defaultStatus)
		}
		_, err := io.WriteString(w, resp.Value)
		return err
	case *runtime.ErrorValue:
		http.Error(w, resp.Message, http.StatusInternalServerError)
		return nil
	case *runtime.ObjectValue:
		status := defaultStatus
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

func buildForwardRequestOptions(name string, inboundValue runtime.Value, optionsValue runtime.Value) (*requestOptions, map[string]string, map[string]struct{}, error) {
	inbound, ok := inboundValue.(*runtime.ObjectValue)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%s expects request object", name)
	}
	options, ok := optionsValue.(*runtime.ObjectValue)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%s expects options object", name)
	}

	urlValue, ok := options.Fields["url"].(runtime.StringValue)
	if !ok || strings.TrimSpace(urlValue.Value) == "" {
		return nil, nil, nil, fmt.Errorf("%s options require non-empty url", name)
	}

	methodValue, ok := inbound.Fields["method"].(runtime.StringValue)
	if !ok || methodValue.Value == "" {
		return nil, nil, nil, fmt.Errorf("%s request requires method", name)
	}
	method := methodValue.Value
	if overrideValue, ok := options.Fields["method"].(runtime.StringValue); ok && overrideValue.Value != "" {
		method = overrideValue.Value
	}
	method = strings.ToUpper(method)

	bodyValue, ok := inbound.Fields["body"].(runtime.StringValue)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%s request requires body", name)
	}
	body := bodyValue.Value
	if overrideValue, ok := options.Fields["body"]; ok {
		body = overrideValue.String()
	}

	headers := make(map[string]string)
	copyHeaders := true
	if copyHeadersValue, ok := options.Fields["copyHeaders"]; ok {
		boolValue, ok := copyHeadersValue.(runtime.BoolValue)
		if !ok {
			return nil, nil, nil, fmt.Errorf("%s copyHeaders must be bool", name)
		}
		copyHeaders = boolValue.Value
	}
	if copyHeaders {
		inboundHeaders, err := httpHeadersFromRuntime(name+" request", inbound.Fields["headers"])
		if err != nil {
			return nil, nil, nil, err
		}
		for key, value := range inboundHeaders {
			if !isHopByHopHeader(key) && !strings.EqualFold(key, "Host") {
				headers[key] = value
			}
		}
	}
	overrideHeaders, err := httpHeadersFromRuntime(name, options.Fields["headers"])
	if err != nil {
		return nil, nil, nil, err
	}
	host := ""
	for key, value := range overrideHeaders {
		if isHopByHopHeader(key) {
			continue
		}
		if strings.EqualFold(key, "Host") {
			host = value
			continue
		}
		headers[key] = value
	}
	stripHeaders, err := httpHeaderNameSet(name, options.Fields["stripHeaders"])
	if err != nil {
		return nil, nil, nil, err
	}
	for key := range stripHeaders {
		delete(headers, key)
	}

	cookies := map[string]string{}
	if inboundCookies, ok := inbound.Fields["cookies"].(*runtime.ObjectValue); ok {
		for key, value := range inboundCookies.Fields {
			if text, ok := value.(runtime.StringValue); ok {
				cookies[key] = text.Value
			}
		}
	}
	overrideCookies, err := httpCookiesFromRuntime(name, options.Fields["cookies"])
	if err != nil {
		return nil, nil, nil, err
	}
	for key, value := range overrideCookies {
		cookies[key] = value
	}

	timeout, err := httpTimeoutFromOptions(name, options)
	if err != nil {
		return nil, nil, nil, err
	}
	followRedirects, maxRedirects, err := httpRedirectOptions(name, options)
	if err != nil {
		return nil, nil, nil, err
	}
	targetURL, err := httpMergeQuery(urlValue.Value, options.Fields["query"])
	if err != nil {
		return nil, nil, nil, err
	}
	responseHeaders, err := httpHeadersFromRuntime(name, options.Fields["responseHeaders"])
	if err != nil {
		return nil, nil, nil, err
	}

	return &requestOptions{
		Method:          method,
		URL:             targetURL,
		Headers:         headers,
		Body:            body,
		Timeout:         timeout,
		Host:            host,
		Cookies:         cookies,
		FollowRedirects: followRedirects,
		MaxRedirects:    maxRedirects,
	}, responseHeaders, stripHeaders, nil
}

func doHTTPRoundTrip(opts *requestOptions) (*http.Response, error) {
	req, err := http.NewRequest(opts.Method, opts.URL, strings.NewReader(opts.Body))
	if err != nil {
		return nil, err
	}
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}
	for key, value := range opts.Cookies {
		req.AddCookie(&http.Cookie{Name: key, Value: value})
	}
	if opts.Host != "" {
		req.Host = opts.Host
	}
	client := &http.Client{Timeout: opts.Timeout}
	if !opts.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else if opts.MaxRedirects > 0 {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= opts.MaxRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		}
	}
	return client.Do(req)
}

func httpHeadersToRuntime(headers http.Header) runtime.Value {
	fields := make(map[string]runtime.Value, len(headers))
	for key, values := range headers {
		fields[key] = httpStringValuesToRuntime(values)
	}
	return &runtime.ObjectValue{Fields: fields}
}

func httpHeadersFromRuntime(name string, value runtime.Value) (map[string]string, error) {
	headers := make(map[string]string)
	if value == nil {
		return headers, nil
	}
	headerObj, ok := value.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s headers must be object", name)
	}
	for key, value := range headerObj.Fields {
		headers[key] = value.String()
	}
	return headers, nil
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

func httpCookieGetter(cookies []*http.Cookie) runtime.Value {
	index := map[string]string{}
	for _, cookie := range cookies {
		index[cookie.Name] = cookie.Value
	}
	return &runtime.NativeFunction{
		Name:  "express.request.cookie",
		Arity: 1,
		Fn: func(args []runtime.Value) (runtime.Value, error) {
			name, err := requireStringArg("cookie", args[0])
			if err != nil {
				return nil, err
			}
			value, ok := index[name]
			if !ok {
				return runtime.NullValue{}, nil
			}
			return runtime.StringValue{Value: value}, nil
		},
	}
}

func httpCookiesToRuntime(cookies []*http.Cookie) runtime.Value {
	fields := make(map[string]runtime.Value, len(cookies))
	for _, cookie := range cookies {
		fields[cookie.Name] = runtime.StringValue{Value: cookie.Value}
	}
	return &runtime.ObjectValue{Fields: fields}
}

func httpCookiesFromRuntime(name string, value runtime.Value) (map[string]string, error) {
	cookies := make(map[string]string)
	if value == nil {
		return cookies, nil
	}
	objectValue, ok := value.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s cookies must be object", name)
	}
	for key, item := range objectValue.Fields {
		text, ok := item.(runtime.StringValue)
		if !ok {
			return nil, fmt.Errorf("%s cookies must be string values", name)
		}
		cookies[key] = text.Value
	}
	return cookies, nil
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

func httpTimeoutFromOptions(name string, obj *runtime.ObjectValue) (time.Duration, error) {
	timeout := 30 * time.Second
	if timeoutValue, ok := obj.Fields["timeoutMs"]; ok {
		intValue, ok := timeoutValue.(runtime.IntValue)
		if !ok {
			return 0, fmt.Errorf("%s timeoutMs must be int", name)
		}
		if intValue.Value < 0 {
			return 0, fmt.Errorf("%s timeoutMs must be non-negative", name)
		}
		timeout = time.Duration(intValue.Value) * time.Millisecond
	}
	return timeout, nil
}

func httpRedirectOptions(name string, obj *runtime.ObjectValue) (bool, int, error) {
	followRedirects := true
	maxRedirects := 0
	if followValue, ok := obj.Fields["followRedirects"]; ok {
		boolValue, ok := followValue.(runtime.BoolValue)
		if !ok {
			return false, 0, fmt.Errorf("%s followRedirects must be bool", name)
		}
		followRedirects = boolValue.Value
	}
	if maxValue, ok := obj.Fields["maxRedirects"]; ok {
		intValue, ok := maxValue.(runtime.IntValue)
		if !ok || intValue.Value < 0 {
			return false, 0, fmt.Errorf("%s maxRedirects must be non-negative int", name)
		}
		maxRedirects = int(intValue.Value)
	}
	return followRedirects, maxRedirects, nil
}

func isHopByHopHeader(header string) bool {
	switch strings.ToLower(header) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

func isCallableValue(value runtime.Value) bool {
	switch value.(type) {
	case *runtime.Closure, *runtime.NativeFunction:
		return true
	default:
		return false
	}
}

func httpRequestBodyToRuntime(contentType string, body []byte) (runtime.Value, runtime.Value, error) {
	formFields := map[string]runtime.Value{}
	fileFields := map[string]runtime.Value{}
	mediaType, params, _ := mime.ParseMediaType(contentType)
	switch {
	case mediaType == "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return &runtime.ObjectValue{Fields: formFields}, &runtime.ObjectValue{Fields: fileFields}, err
		}
		for key, items := range values {
			formFields[key] = httpStringValuesToRuntime(items)
		}
	case strings.HasPrefix(mediaType, "multipart/"):
		boundary := params["boundary"]
		reader := multipart.NewReader(bytes.NewReader(body), boundary)
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return &runtime.ObjectValue{Fields: formFields}, &runtime.ObjectValue{Fields: fileFields}, err
			}
			data, err := io.ReadAll(part)
			if err != nil {
				return &runtime.ObjectValue{Fields: formFields}, &runtime.ObjectValue{Fields: fileFields}, err
			}
			if part.FileName() == "" {
				httpAppendRuntimeField(formFields, part.FormName(), runtime.StringValue{Value: string(data)})
				continue
			}
			contentTypeValue := part.Header.Get("Content-Type")
			if contentTypeValue == "" {
				contentTypeValue = http.DetectContentType(data)
			}
			fileObject := &runtime.ObjectValue{Fields: map[string]runtime.Value{
				"field":       runtime.StringValue{Value: part.FormName()},
				"filename":    runtime.StringValue{Value: part.FileName()},
				"contentType": runtime.StringValue{Value: contentTypeValue},
				"size":        runtime.IntValue{Value: int64(len(data))},
				"text":        runtime.StringValue{Value: string(data)},
			}}
			existing, ok := fileFields[part.FormName()].(*runtime.ArrayValue)
			if !ok {
				fileFields[part.FormName()] = &runtime.ArrayValue{Elements: []runtime.Value{fileObject}}
				continue
			}
			existing.Elements = append(existing.Elements, fileObject)
		}
	}
	return &runtime.ObjectValue{Fields: formFields}, &runtime.ObjectValue{Fields: fileFields}, nil
}

func httpAppendRuntimeField(fields map[string]runtime.Value, key string, value runtime.Value) {
	if existing, ok := fields[key]; ok {
		if arrayValue, ok := existing.(*runtime.ArrayValue); ok {
			arrayValue.Elements = append(arrayValue.Elements, value)
			return
		}
		fields[key] = &runtime.ArrayValue{Elements: []runtime.Value{existing, value}}
		return
	}
	fields[key] = value
}

func httpFileGetter(filesValue runtime.Value) runtime.Value {
	return &runtime.NativeFunction{
		Name:  "express.request.file",
		Arity: 1,
		Fn: func(args []runtime.Value) (runtime.Value, error) {
			name, err := requireStringArg("file", args[0])
			if err != nil {
				return nil, err
			}
			filesObject, ok := filesValue.(*runtime.ObjectValue)
			if !ok {
				return runtime.NullValue{}, nil
			}
			entry, ok := filesObject.Fields[name]
			if !ok {
				return runtime.NullValue{}, nil
			}
			arrayValue, ok := entry.(*runtime.ArrayValue)
			if !ok || len(arrayValue.Elements) == 0 {
				return runtime.NullValue{}, nil
			}
			return arrayValue.Elements[0], nil
		},
	}
}

func httpCookieFromRuntime(name string, value string, extra []runtime.Value) (*http.Cookie, error) {
	cookie := &http.Cookie{Name: name, Value: value, Path: "/"}
	if len(extra) == 0 {
		return cookie, nil
	}
	options, ok := extra[0].(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("cookie options must be object")
	}
	if pathValue, ok := options.Fields["path"]; ok {
		path, err := requireStringArg("cookie", pathValue)
		if err != nil {
			return nil, err
		}
		cookie.Path = path
	}
	if domainValue, ok := options.Fields["domain"]; ok {
		domain, err := requireStringArg("cookie", domainValue)
		if err != nil {
			return nil, err
		}
		cookie.Domain = domain
	}
	if secureValue, ok := options.Fields["secure"]; ok {
		boolValue, ok := secureValue.(runtime.BoolValue)
		if !ok {
			return nil, fmt.Errorf("cookie secure must be bool")
		}
		cookie.Secure = boolValue.Value
	}
	if httpOnlyValue, ok := options.Fields["httpOnly"]; ok {
		boolValue, ok := httpOnlyValue.(runtime.BoolValue)
		if !ok {
			return nil, fmt.Errorf("cookie httpOnly must be bool")
		}
		cookie.HttpOnly = boolValue.Value
	}
	if maxAgeValue, ok := options.Fields["maxAge"]; ok {
		intValue, ok := maxAgeValue.(runtime.IntValue)
		if !ok {
			return nil, fmt.Errorf("cookie maxAge must be int")
		}
		cookie.MaxAge = int(intValue.Value)
	}
	return cookie, nil
}

func httpHeaderNameSet(name string, value runtime.Value) (map[string]struct{}, error) {
	set := map[string]struct{}{}
	if value == nil {
		return set, nil
	}
	arrayValue, ok := value.(*runtime.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("%s stripHeaders must be array", name)
	}
	for _, item := range arrayValue.Elements {
		header, err := requireStringArg(name, item)
		if err != nil {
			return nil, err
		}
		set[strings.ToLower(header)] = struct{}{}
	}
	return set, nil
}

func httpMergeQuery(target string, value runtime.Value) (string, error) {
	if value == nil {
		return target, nil
	}
	queryObject, ok := value.(*runtime.ObjectValue)
	if !ok {
		return "", fmt.Errorf("query must be object")
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	for key, fieldValue := range queryObject.Fields {
		query.Del(key)
		switch typed := fieldValue.(type) {
		case *runtime.ArrayValue:
			for _, item := range typed.Elements {
				query.Add(key, item.String())
			}
		default:
			query.Set(key, fieldValue.String())
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func httpStringValuesToRuntime(values []string) runtime.Value {
	if len(values) == 1 {
		return runtime.StringValue{Value: values[0]}
	}
	items := make([]runtime.Value, 0, len(values))
	for _, value := range values {
		items = append(items, runtime.StringValue{Value: value})
	}
	return &runtime.ArrayValue{Elements: items}
}

func requireStringArg(name string, v runtime.Value) (string, error) {
	text, ok := v.(runtime.StringValue)
	if !ok {
		return "", fmt.Errorf("%s expects string argument", name)
	}
	return text.Value, nil
}
