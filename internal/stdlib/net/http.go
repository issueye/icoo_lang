package stdnet

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

func LoadStdNetHTTPClientModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.http.client",
		Path: "std.http.client",
		Exports: map[string]runtime.Value{
			"delete":      &runtime.NativeFunction{Name: "delete", Arity: 1, Fn: httpDelete},
			"download":    &runtime.NativeFunction{Name: "download", Arity: 2, Fn: httpDownload},
			"get":         &runtime.NativeFunction{Name: "get", Arity: 1, Fn: httpGet},
			"getJSON":     &runtime.NativeFunction{Name: "getJSON", Arity: 1, Fn: httpGetJSON},
			"post":        &runtime.NativeFunction{Name: "post", Arity: 2, Fn: httpPost},
			"put":         &runtime.NativeFunction{Name: "put", Arity: 2, Fn: httpPut},
			"request":     &runtime.NativeFunction{Name: "request", Arity: 1, Fn: httpRequest},
			"requestJSON": &runtime.NativeFunction{Name: "requestJSON", Arity: 1, Fn: httpRequestJSON},
		},
		Done: true,
	}
}

func LoadStdNetHTTPServerModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.http.server",
		Path: "std.http.server",
		Exports: map[string]runtime.Value{
			"forward": &runtime.NativeFunction{Name: "forward", Arity: 2, Fn: httpForward},
			"listen":  &runtime.NativeFunction{Name: "listen", Arity: 1, CtxFn: httpListen},
		},
		Done: true,
	}
}

func httpGet(args []runtime.Value) (runtime.Value, error) {
	url, err := utils.RequireStringArg("get", args[0])
	if err != nil {
		return nil, err
	}
	return doHTTPRequest(&httpRequestOptions{
		Method: "GET",
		URL:    url,
	})
}

func httpGetJSON(args []runtime.Value) (runtime.Value, error) {
	url, err := utils.RequireStringArg("getJSON", args[0])
	if err != nil {
		return nil, err
	}
	return doHTTPRequest(&httpRequestOptions{
		Method:     "GET",
		URL:        url,
		ExpectJSON: true,
	})
}

func httpPost(args []runtime.Value) (runtime.Value, error) {
	return httpSimpleBodyRequest("post", "POST", args)
}

func httpPut(args []runtime.Value) (runtime.Value, error) {
	return httpSimpleBodyRequest("put", "PUT", args)
}

func httpDelete(args []runtime.Value) (runtime.Value, error) {
	url, err := utils.RequireStringArg("delete", args[0])
	if err != nil {
		return nil, err
	}
	return doHTTPRequest(&httpRequestOptions{
		Method: "DELETE",
		URL:    url,
	})
}

func httpRequest(args []runtime.Value) (runtime.Value, error) {
	opts, err := parseHTTPRequestOptions(args[0])
	if err != nil {
		return nil, err
	}
	return doHTTPRequest(opts)
}

func httpRequestJSON(args []runtime.Value) (runtime.Value, error) {
	opts, err := parseHTTPRequestOptions(args[0])
	if err != nil {
		return nil, err
	}
	obj := args[0].(*runtime.ObjectValue)
	if jsonBodyValue, ok := obj.Fields["json"]; ok {
		plain, err := utils.RuntimeToPlainValue(jsonBodyValue)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(plain)
		if err != nil {
			return nil, err
		}
		opts.Body = string(data)
		if opts.Headers["Content-Type"] == "" {
			opts.Headers["Content-Type"] = "application/json"
		}
	}
	if opts.Headers["Accept"] == "" {
		opts.Headers["Accept"] = "application/json"
	}
	opts.ExpectJSON = true
	return doHTTPRequest(opts)
}

func httpDownload(args []runtime.Value) (runtime.Value, error) {
	url, err := utils.RequireStringArg("download", args[0])
	if err != nil {
		return nil, err
	}
	path, err := utils.RequireStringArg("download", args[1])
	if err != nil {
		return nil, err
	}

	respValue, err := doHTTPRequest(&httpRequestOptions{
		Method: "GET",
		URL:    url,
	})
	if err != nil {
		return nil, err
	}
	respObj := respValue.(*runtime.ObjectValue)
	bodyText, _ := respObj.Fields["body"].(runtime.StringValue)
	if err := os.WriteFile(path, []byte(bodyText.Value), 0o644); err != nil {
		return nil, err
	}
	respObj.Fields["path"] = runtime.StringValue{Value: path}
	return respObj, nil
}

type httpServerBinding struct {
	server *http.Server
}

type httpResponseBinding struct {
	writer      http.ResponseWriter
	flusher     http.Flusher
	statusCode  int
	wroteHeader bool
	handled     bool
	handle      *runtime.ObjectValue
}

type httpRequestOptions struct {
	Method     string
	URL        string
	Headers    map[string]string
	Body       string
	Timeout    time.Duration
	ExpectJSON bool
	Host       string
}

func parseHTTPRequestOptions(v runtime.Value) (*httpRequestOptions, error) {
	obj, ok := v.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("request expects options object")
	}

	urlValue, ok := obj.Fields["url"].(runtime.StringValue)
	if !ok || strings.TrimSpace(urlValue.Value) == "" {
		return nil, fmt.Errorf("request options require non-empty url")
	}

	method := "GET"
	if methodValue, ok := obj.Fields["method"].(runtime.StringValue); ok && methodValue.Value != "" {
		method = strings.ToUpper(methodValue.Value)
	}

	headers, err := httpHeadersFromRuntime("request", obj.Fields["headers"])
	if err != nil {
		return nil, err
	}

	body := ""
	if bodyValue, ok := obj.Fields["body"]; ok {
		body = bodyValue.String()
	}

	timeout, err := httpTimeoutFromOptions("request", obj)
	if err != nil {
		return nil, err
	}

	return &httpRequestOptions{
		Method:  method,
		URL:     urlValue.Value,
		Headers: headers,
		Body:    body,
		Timeout: timeout,
	}, nil
}

func doHTTPRequest(opts *httpRequestOptions) (runtime.Value, error) {
	resp, err := doHTTPRoundTrip(opts)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	result := &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"ok":            runtime.BoolValue{Value: resp.StatusCode >= 200 && resp.StatusCode < 300},
		"status":        runtime.IntValue{Value: int64(resp.StatusCode)},
		"statusText":    runtime.StringValue{Value: resp.Status},
		"body":          runtime.StringValue{Value: string(body)},
		"header":        httpHeaderGetter(resp.Header),
		"hasHeader":     httpHasHeaderGetter(resp.Header),
		"url":           runtime.StringValue{Value: resp.Request.URL.String()},
		"method":        runtime.StringValue{Value: opts.Method},
		"headers":       httpHeadersToRuntime(resp.Header),
		"contentLength": runtime.IntValue{Value: resp.ContentLength},
	}}
	if opts.ExpectJSON && len(body) > 0 {
		var decoded any
		if err := json.Unmarshal(body, &decoded); err != nil {
			return nil, err
		}
		result.Fields["json"] = utils.PlainToRuntimeValue(decoded)
	}
	return result, nil
}

func buildForwardRequestOptions(name string, inboundValue runtime.Value, optionsValue runtime.Value) (*httpRequestOptions, error) {
	inbound, ok := inboundValue.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s expects request object", name)
	}
	options, ok := optionsValue.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s expects options object", name)
	}

	urlValue, ok := options.Fields["url"].(runtime.StringValue)
	if !ok || strings.TrimSpace(urlValue.Value) == "" {
		return nil, fmt.Errorf("%s options require non-empty url", name)
	}

	methodValue, ok := inbound.Fields["method"].(runtime.StringValue)
	if !ok || methodValue.Value == "" {
		return nil, fmt.Errorf("%s request requires method", name)
	}
	method := methodValue.Value
	if overrideValue, ok := options.Fields["method"].(runtime.StringValue); ok && overrideValue.Value != "" {
		method = overrideValue.Value
	}
	method = strings.ToUpper(method)

	bodyValue, ok := inbound.Fields["body"].(runtime.StringValue)
	if !ok {
		return nil, fmt.Errorf("%s request requires body", name)
	}
	body := bodyValue.Value
	if overrideValue, ok := options.Fields["body"]; ok {
		body = overrideValue.String()
	}

	headers := make(map[string]string)
	inboundHeaders, err := httpHeadersFromRuntime(name+" request", inbound.Fields["headers"])
	if err != nil {
		return nil, err
	}
	for key, value := range inboundHeaders {
		if !isHopByHopHeader(key) && !strings.EqualFold(key, "Host") {
			headers[key] = value
		}
	}
	overrideHeaders, err := httpHeadersFromRuntime(name, options.Fields["headers"])
	if err != nil {
		return nil, err
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

	timeout, err := httpTimeoutFromOptions(name, options)
	if err != nil {
		return nil, err
	}

	return &httpRequestOptions{
		Method:  method,
		URL:     urlValue.Value,
		Headers: headers,
		Body:    body,
		Timeout: timeout,
		Host:    host,
	}, nil
}

func doHTTPRoundTrip(opts *httpRequestOptions) (*http.Response, error) {
	req, err := http.NewRequest(opts.Method, opts.URL, strings.NewReader(opts.Body))
	if err != nil {
		return nil, err
	}
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	if opts.Host != "" {
		req.Host = opts.Host
	}

	client := &http.Client{Timeout: opts.Timeout}
	return client.Do(req)
}

func httpForward(args []runtime.Value) (runtime.Value, error) {
	opts, err := buildForwardRequestOptions("forward", args[0], args[1])
	if err != nil {
		return nil, err
	}
	return doHTTPRequest(opts)
}

func httpListen(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
	obj, ok := args[0].(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("listen expects options object")
	}
	addrValue, ok := obj.Fields["addr"].(runtime.StringValue)
	if !ok || strings.TrimSpace(addrValue.Value) == "" {
		return nil, fmt.Errorf("listen options require non-empty addr")
	}
	handlerValue, ok := obj.Fields["handler"]
	if !ok {
		return nil, fmt.Errorf("listen options require handler")
	}
	if !isCallableValue(handlerValue) {
		return nil, fmt.Errorf("listen handler must be callable")
	}

	binding := &httpServerBinding{}
	listener, err := net.Listen("tcp", addrValue.Value)
	if err != nil {
		return nil, err
	}

	binding.server = &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			respBinding := newHTTPResponseHandle(w)
			callArgs := []runtime.Value{reqValue}
			if closure, ok := handlerValue.(*runtime.Closure); ok && closure.Proto != nil && closure.Proto.Arity == 2 {
				callArgs = []runtime.Value{reqValue, respBinding.handle}
			} else if native, ok := handlerValue.(*runtime.NativeFunction); ok && native.Arity == 2 {
				callArgs = []runtime.Value{reqValue, respBinding.handle}
			}

			respValue, err := ctx.CallDetached(handlerValue, callArgs)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if respBinding.handled {
				return
			}
			if err := writeHTTPServerResponse(w, respValue, respBinding.statusCode); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}),
	}

	go func() {
		_ = binding.server.Serve(listener)
	}()

	addr := listener.Addr().String()
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"addr":  runtime.StringValue{Value: addr},
		"url":   runtime.StringValue{Value: "http://" + addr},
		"close": &runtime.NativeFunction{Name: "close", Arity: 0, Fn: binding.close},
	}}, nil
}

func newHTTPResponseHandle(w http.ResponseWriter) *httpResponseBinding {
	flusher, _ := w.(http.Flusher)
	binding := &httpResponseBinding{
		writer:     w,
		flusher:    flusher,
		statusCode: http.StatusOK,
	}
	binding.handle = &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"proxy":     &runtime.NativeFunction{Name: "response.proxy", Arity: 2, Fn: binding.proxy},
		"statusCode": &runtime.NativeFunction{Name: "response.statusCode", Arity: 0, Fn: binding.statusCodeValue},
		"status":    &runtime.NativeFunction{Name: "response.status", Arity: 1, Fn: binding.status},
		"setHeader": &runtime.NativeFunction{Name: "response.setHeader", Arity: 2, Fn: binding.setHeader},
		"sse":       &runtime.NativeFunction{Name: "response.sse", Arity: 1, Fn: binding.writeSSE},
		"write":     &runtime.NativeFunction{Name: "response.write", Arity: 1, Fn: binding.write},
		"json":      &runtime.NativeFunction{Name: "response.json", Arity: 1, Fn: binding.writeJSON},
		"flush":     &runtime.NativeFunction{Name: "response.flush", Arity: 0, Fn: binding.flush},
		"end":       &runtime.NativeFunction{Name: "response.end", Arity: -1, Fn: binding.end},
	}}
	return binding
}

func (binding *httpResponseBinding) statusCodeValue(args []runtime.Value) (runtime.Value, error) {
	return runtime.IntValue{Value: int64(binding.statusCode)}, nil
}

func (binding *httpResponseBinding) status(args []runtime.Value) (runtime.Value, error) {
	code, ok := args[0].(runtime.IntValue)
	if !ok {
		return nil, fmt.Errorf("response.status expects int argument")
	}
	binding.statusCode = int(code.Value)
	return binding.handle, nil
}

func (binding *httpResponseBinding) setHeader(args []runtime.Value) (runtime.Value, error) {
	name, err := utils.RequireStringArg("response.setHeader", args[0])
	if err != nil {
		return nil, err
	}
	binding.writer.Header().Set(name, args[1].String())
	return binding.handle, nil
}

func (binding *httpResponseBinding) write(args []runtime.Value) (runtime.Value, error) {
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

func (binding *httpResponseBinding) writeSSE(args []runtime.Value) (runtime.Value, error) {
	if binding.writer.Header().Get("Content-Type") == "" {
		binding.writer.Header().Set("Content-Type", "text/event-stream")
	}
	if err := binding.ensureHeader(); err != nil {
		return nil, err
	}
	binding.handled = true
	text, err := formatSSEEvent(args[0])
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

func (binding *httpResponseBinding) writeJSON(args []runtime.Value) (runtime.Value, error) {
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

func (binding *httpResponseBinding) flush(args []runtime.Value) (runtime.Value, error) {
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

func (binding *httpResponseBinding) end(args []runtime.Value) (runtime.Value, error) {
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

func (binding *httpResponseBinding) proxy(args []runtime.Value) (runtime.Value, error) {
	opts, err := buildForwardRequestOptions("response.proxy", args[0], args[1])
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
		binding.writer.Header().Del(key)
		for _, value := range values {
			binding.writer.Header().Add(key, value)
		}
	}

	binding.statusCode = resp.StatusCode
	if err := binding.ensureHeader(); err != nil {
		return nil, err
	}
	binding.handled = true

	target := io.Writer(binding.writer)
	if binding.flusher != nil {
		target = &httpFlushingWriter{
			writer:  binding.writer,
			flusher: binding.flusher,
		}
	}
	if _, err := io.Copy(target, resp.Body); err != nil {
		return nil, err
	}
	return binding.handle, nil
}

func (binding *httpResponseBinding) ensureHeader() error {
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

type httpFlushingWriter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

func (w *httpFlushingWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if err == nil && n > 0 && w.flusher != nil {
		w.flusher.Flush()
	}
	return n, err
}

func (binding *httpServerBinding) close(args []runtime.Value) (runtime.Value, error) {
	if binding == nil || binding.server == nil {
		return runtime.NullValue{}, nil
	}
	return runtime.NullValue{}, binding.server.Close()
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
		if len(values) == 1 {
			queryFields[key] = runtime.StringValue{Value: values[0]}
			continue
		}
		items := make([]runtime.Value, len(values))
		for i, value := range values {
			items[i] = runtime.StringValue{Value: value}
		}
		queryFields[key] = &runtime.ArrayValue{Elements: items}
	}

	req := &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"method":        runtime.StringValue{Value: r.Method},
		"url":           runtime.StringValue{Value: r.URL.String()},
		"path":          runtime.StringValue{Value: r.URL.Path},
		"query":         &runtime.ObjectValue{Fields: queryFields},
		"header":        httpHeaderGetter(r.Header),
		"hasHeader":     httpHasHeaderGetter(r.Header),
		"headers":       httpHeadersToRuntime(r.Header),
		"body":          runtime.StringValue{Value: string(body)},
		"contentLength": runtime.IntValue{Value: r.ContentLength},
		"host":          runtime.StringValue{Value: r.Host},
		"remoteAddr":    runtime.StringValue{Value: r.RemoteAddr},
		"requestId":     runtime.StringValue{Value: requestID},
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

func writeHTTPServerResponse(w http.ResponseWriter, value runtime.Value, defaultStatus int) error {
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
		return fmt.Errorf("unsupported http response value: %s", runtime.KindName(value))
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
		Name:  "http.header",
		Arity: 1,
		Fn: func(args []runtime.Value) (runtime.Value, error) {
			name, err := utils.RequireStringArg("header", args[0])
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
		Name:  "http.hasHeader",
		Arity: 1,
		Fn: func(args []runtime.Value) (runtime.Value, error) {
			name, err := utils.RequireStringArg("hasHeader", args[0])
			if err != nil {
				return nil, err
			}
			return runtime.BoolValue{Value: headers.Get(name) != ""}, nil
		},
	}
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

func isHopByHopHeader(header string) bool {
	switch strings.ToLower(header) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

func httpSimpleBodyRequest(name string, method string, args []runtime.Value) (runtime.Value, error) {
	url, err := utils.RequireStringArg(name, args[0])
	if err != nil {
		return nil, err
	}
	body, err := utils.RequireStringArg(name, args[1])
	if err != nil {
		return nil, err
	}
	return doHTTPRequest(&httpRequestOptions{
		Method: method,
		URL:    url,
		Body:   body,
	})
}
