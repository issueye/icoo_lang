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
	resp, err := client.Do(req)
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

func httpForward(args []runtime.Value) (runtime.Value, error) {
	inbound, ok := args[0].(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("forward expects request object")
	}
	options, ok := args[1].(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("forward expects options object")
	}

	urlValue, ok := options.Fields["url"].(runtime.StringValue)
	if !ok || strings.TrimSpace(urlValue.Value) == "" {
		return nil, fmt.Errorf("forward options require non-empty url")
	}

	methodValue, ok := inbound.Fields["method"].(runtime.StringValue)
	if !ok || methodValue.Value == "" {
		return nil, fmt.Errorf("forward request requires method")
	}
	method := methodValue.Value
	if overrideValue, ok := options.Fields["method"].(runtime.StringValue); ok && overrideValue.Value != "" {
		method = overrideValue.Value
	}
	method = strings.ToUpper(method)

	bodyValue, ok := inbound.Fields["body"].(runtime.StringValue)
	if !ok {
		return nil, fmt.Errorf("forward request requires body")
	}
	body := bodyValue.Value
	if overrideValue, ok := options.Fields["body"]; ok {
		body = overrideValue.String()
	}

	headers := make(map[string]string)
	inboundHeaders, err := httpHeadersFromRuntime("forward request", inbound.Fields["headers"])
	if err != nil {
		return nil, err
	}
	for key, value := range inboundHeaders {
		if !isHopByHopHeader(key) && !strings.EqualFold(key, "Host") {
			headers[key] = value
		}
	}
	overrideHeaders, err := httpHeadersFromRuntime("forward", options.Fields["headers"])
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

	timeout, err := httpTimeoutFromOptions("forward", options)
	if err != nil {
		return nil, err
	}

	return doHTTPRequest(&httpRequestOptions{
		Method:  method,
		URL:     urlValue.Value,
		Headers: headers,
		Body:    body,
		Timeout: timeout,
		Host:    host,
	})
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

			respValue, err := ctx.CallDetached(handlerValue, []runtime.Value{reqValue})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := writeHTTPServerResponse(w, respValue); err != nil {
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

	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"method":        runtime.StringValue{Value: r.Method},
		"url":           runtime.StringValue{Value: r.URL.String()},
		"path":          runtime.StringValue{Value: r.URL.Path},
		"query":         &runtime.ObjectValue{Fields: queryFields},
		"headers":       httpHeadersToRuntime(r.Header),
		"body":          runtime.StringValue{Value: string(body)},
		"contentLength": runtime.IntValue{Value: r.ContentLength},
		"host":          runtime.StringValue{Value: r.Host},
		"remoteAddr":    runtime.StringValue{Value: r.RemoteAddr},
	}}, nil
}

func writeHTTPServerResponse(w http.ResponseWriter, value runtime.Value) error {
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
