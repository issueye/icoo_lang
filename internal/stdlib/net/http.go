package stdnet

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
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
			"delete":        &runtime.NativeFunction{Name: "delete", Arity: 1, Fn: httpDelete},
			"download":      &runtime.NativeFunction{Name: "download", Arity: 2, Fn: httpDownload},
			"get":           &runtime.NativeFunction{Name: "get", Arity: 1, Fn: httpGet},
			"getJSON":       &runtime.NativeFunction{Name: "getJSON", Arity: 1, Fn: httpGetJSON},
			"post":          &runtime.NativeFunction{Name: "post", Arity: 2, Fn: httpPost},
			"put":           &runtime.NativeFunction{Name: "put", Arity: 2, Fn: httpPut},
			"request":       &runtime.NativeFunction{Name: "request", Arity: 1, Fn: httpRequest},
			"requestJSON":   &runtime.NativeFunction{Name: "requestJSON", Arity: 1, Fn: httpRequestJSON},
			"requestStream": &runtime.NativeFunction{Name: "requestStream", Arity: 1, Fn: httpRequestStream},
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

type httpStreamBinding struct {
	body   io.ReadCloser
	reader *bufio.Reader
}

type httpRequestOptions struct {
	Method          string
	URL             string
	Headers         map[string]string
	Body            string
	Timeout         time.Duration
	ExpectJSON      bool
	Host            string
	Cookies         map[string]string
	FollowRedirects bool
	MaxRedirects    int
}

func httpGet(args []runtime.Value) (runtime.Value, error) {
	url, err := utils.RequireStringArg("get", args[0])
	if err != nil {
		return nil, err
	}
	return doHTTPRequest(&httpRequestOptions{
		Method:          "GET",
		URL:             url,
		FollowRedirects: true,
	})
}

func httpGetJSON(args []runtime.Value) (runtime.Value, error) {
	url, err := utils.RequireStringArg("getJSON", args[0])
	if err != nil {
		return nil, err
	}
	return doHTTPRequest(&httpRequestOptions{
		Method:          "GET",
		URL:             url,
		ExpectJSON:      true,
		FollowRedirects: true,
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
		Method:          "DELETE",
		URL:             url,
		FollowRedirects: true,
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

func httpRequestStream(args []runtime.Value) (runtime.Value, error) {
	opts, err := parseHTTPRequestOptions(args[0])
	if err != nil {
		return nil, err
	}
	resp, err := doHTTPRoundTrip(opts)
	if err != nil {
		return nil, err
	}
	return newHTTPStreamObject(resp, opts), nil
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
		Method:          "GET",
		URL:             url,
		FollowRedirects: true,
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
	cookies, err := httpCookiesFromRuntime("request", obj.Fields["cookies"])
	if err != nil {
		return nil, err
	}

	body, contentType, err := httpBodyFromOptions("request", obj)
	if err != nil {
		return nil, err
	}
	if contentType != "" && headers["Content-Type"] == "" {
		headers["Content-Type"] = contentType
	}

	timeout, err := httpTimeoutFromOptions("request", obj)
	if err != nil {
		return nil, err
	}
	followRedirects, maxRedirects, err := httpRedirectOptions("request", obj)
	if err != nil {
		return nil, err
	}

	return &httpRequestOptions{
		Method:          method,
		URL:             urlValue.Value,
		Headers:         headers,
		Body:            body,
		Timeout:         timeout,
		Cookies:         cookies,
		FollowRedirects: followRedirects,
		MaxRedirects:    maxRedirects,
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
	result := httpResponseToRuntime(resp, opts, string(body))
	if opts.ExpectJSON && len(body) > 0 {
		var decoded any
		if err := json.Unmarshal(body, &decoded); err != nil {
			return nil, err
		}
		result.Fields["json"] = utils.PlainToRuntimeValue(decoded)
	}
	return result, nil
}

func httpResponseToRuntime(resp *http.Response, opts *httpRequestOptions, body string) *runtime.ObjectValue {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"ok":            runtime.BoolValue{Value: resp.StatusCode >= 200 && resp.StatusCode < 300},
		"status":        runtime.IntValue{Value: int64(resp.StatusCode)},
		"statusText":    runtime.StringValue{Value: resp.Status},
		"body":          runtime.StringValue{Value: body},
		"header":        httpHeaderGetter(resp.Header),
		"hasHeader":     httpHasHeaderGetter(resp.Header),
		"cookie":        httpCookieGetter(resp.Cookies()),
		"url":           runtime.StringValue{Value: resp.Request.URL.String()},
		"method":        runtime.StringValue{Value: opts.Method},
		"headers":       httpHeadersToRuntime(resp.Header),
		"cookies":       httpCookiesToRuntime(resp.Cookies()),
		"contentLength": runtime.IntValue{Value: resp.ContentLength},
	}}
}

func newHTTPStreamObject(resp *http.Response, opts *httpRequestOptions) *runtime.ObjectValue {
	binding := &httpStreamBinding{
		body:   resp.Body,
		reader: bufio.NewReader(resp.Body),
	}
	obj := httpResponseToRuntime(resp, opts, "")
	obj.Fields["read"] = &runtime.NativeFunction{Name: "http.stream.read", Arity: -1, Fn: binding.read}
	obj.Fields["readAll"] = &runtime.NativeFunction{Name: "http.stream.readAll", Arity: 0, Fn: binding.readAll}
	obj.Fields["close"] = &runtime.NativeFunction{Name: "http.stream.close", Arity: 0, Fn: binding.close}
	return obj
}

func (binding *httpStreamBinding) read(args []runtime.Value) (runtime.Value, error) {
	size := 4096
	if len(args) > 1 {
		return nil, fmt.Errorf("read expects optional size")
	}
	if len(args) == 1 {
		intValue, ok := args[0].(runtime.IntValue)
		if !ok || intValue.Value <= 0 {
			return nil, fmt.Errorf("read size must be positive int")
		}
		size = int(intValue.Value)
	}
	buffer := make([]byte, size)
	n, err := binding.reader.Read(buffer)
	if err == io.EOF && n == 0 {
		return runtime.NullValue{}, nil
	}
	if err != nil && err != io.EOF {
		return nil, err
	}
	return runtime.StringValue{Value: string(buffer[:n])}, nil
}

func (binding *httpStreamBinding) readAll(args []runtime.Value) (runtime.Value, error) {
	data, err := io.ReadAll(binding.reader)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(data)}, nil
}

func (binding *httpStreamBinding) close(args []runtime.Value) (runtime.Value, error) {
	if binding.body == nil {
		return runtime.NullValue{}, nil
	}
	err := binding.body.Close()
	binding.body = nil
	return runtime.NullValue{}, err
}

func buildForwardRequestOptions(name string, inboundValue runtime.Value, optionsValue runtime.Value) (*httpRequestOptions, map[string]string, map[string]struct{}, error) {
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
	return &httpRequestOptions{
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

func doHTTPRoundTrip(opts *httpRequestOptions) (*http.Response, error) {
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

func httpForward(args []runtime.Value) (runtime.Value, error) {
	opts, _, _, err := buildForwardRequestOptions("forward", args[0], args[1])
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
		"proxy":       &runtime.NativeFunction{Name: "response.proxy", Arity: 2, Fn: binding.proxy},
		"statusCode":  &runtime.NativeFunction{Name: "response.statusCode", Arity: 0, Fn: binding.statusCodeValue},
		"status":      &runtime.NativeFunction{Name: "response.status", Arity: 1, Fn: binding.status},
		"setHeader":   &runtime.NativeFunction{Name: "response.setHeader", Arity: 2, Fn: binding.setHeader},
		"setCookie":   &runtime.NativeFunction{Name: "response.setCookie", Arity: -1, Fn: binding.setCookie},
		"clearCookie": &runtime.NativeFunction{Name: "response.clearCookie", Arity: -1, Fn: binding.clearCookie},
		"sse":         &runtime.NativeFunction{Name: "response.sse", Arity: 1, Fn: binding.writeSSE},
		"write":       &runtime.NativeFunction{Name: "response.write", Arity: 1, Fn: binding.write},
		"json":        &runtime.NativeFunction{Name: "response.json", Arity: 1, Fn: binding.writeJSON},
		"flush":       &runtime.NativeFunction{Name: "response.flush", Arity: 0, Fn: binding.flush},
		"end":         &runtime.NativeFunction{Name: "response.end", Arity: -1, Fn: binding.end},
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

func (binding *httpResponseBinding) setCookie(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("response.setCookie expects name, value, and optional options")
	}
	name, err := utils.RequireStringArg("response.setCookie", args[0])
	if err != nil {
		return nil, err
	}
	value, err := utils.RequireStringArg("response.setCookie", args[1])
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

func (binding *httpResponseBinding) clearCookie(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("response.clearCookie expects name and optional options")
	}
	name, err := utils.RequireStringArg("response.clearCookie", args[0])
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
		queryFields[key] = httpStringValuesToRuntime(values)
	}

	formValue, filesValue, err := httpRequestBodyToRuntime(r.Header.Get("Content-Type"), body)
	if err != nil {
		return nil, err
	}
	req := &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"method":        runtime.StringValue{Value: r.Method},
		"url":           runtime.StringValue{Value: r.URL.String()},
		"path":          runtime.StringValue{Value: r.URL.Path},
		"query":         &runtime.ObjectValue{Fields: queryFields},
		"header":        httpHeaderGetter(r.Header),
		"hasHeader":     httpHasHeaderGetter(r.Header),
		"headers":       httpHeadersToRuntime(r.Header),
		"cookie":        httpCookieGetter(r.Cookies()),
		"cookies":       httpCookiesToRuntime(r.Cookies()),
		"body":          runtime.StringValue{Value: string(body)},
		"contentLength": runtime.IntValue{Value: r.ContentLength},
		"host":          runtime.StringValue{Value: r.Host},
		"remoteAddr":    runtime.StringValue{Value: r.RemoteAddr},
		"requestId":     runtime.StringValue{Value: requestID},
		"form":          formValue,
		"files":         filesValue,
	}}
	req.Fields["file"] = httpFileGetter(filesValue)
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

func httpCookieGetter(cookies []*http.Cookie) runtime.Value {
	index := map[string]string{}
	for _, cookie := range cookies {
		index[cookie.Name] = cookie.Value
	}
	return &runtime.NativeFunction{
		Name:  "http.cookie",
		Arity: 1,
		Fn: func(args []runtime.Value) (runtime.Value, error) {
			name, err := utils.RequireStringArg("cookie", args[0])
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
		Method:          method,
		URL:             url,
		Body:            body,
		FollowRedirects: true,
	})
}

func httpBodyFromOptions(name string, obj *runtime.ObjectValue) (string, string, error) {
	hasBody := false
	hasForm := false
	hasMultipart := false
	if _, ok := obj.Fields["body"]; ok {
		hasBody = true
	}
	if _, ok := obj.Fields["form"]; ok {
		hasForm = true
	}
	if _, ok := obj.Fields["multipart"]; ok || obj.Fields["files"] != nil {
		hasMultipart = true
	}
	if boolCount(hasBody)+boolCount(hasForm)+boolCount(hasMultipart) > 1 {
		return "", "", fmt.Errorf("%s body/form/multipart are mutually exclusive", name)
	}
	if hasBody {
		return obj.Fields["body"].String(), "", nil
	}
	if hasForm {
		values, err := httpFormValuesFromRuntime(name, obj.Fields["form"])
		if err != nil {
			return "", "", err
		}
		return values.Encode(), "application/x-www-form-urlencoded", nil
	}
	if hasMultipart {
		return httpMultipartBodyFromRuntime(name, obj.Fields["multipart"], obj.Fields["files"])
	}
	return "", "", nil
}

func httpFormValuesFromRuntime(name string, value runtime.Value) (url.Values, error) {
	values := url.Values{}
	objectValue, ok := value.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s form must be object", name)
	}
	for key, fieldValue := range objectValue.Fields {
		switch typed := fieldValue.(type) {
		case *runtime.ArrayValue:
			for _, item := range typed.Elements {
				values.Add(key, item.String())
			}
		default:
			values.Add(key, fieldValue.String())
		}
	}
	return values, nil
}

func httpMultipartBodyFromRuntime(name string, fieldsValue runtime.Value, filesValue runtime.Value) (string, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if fieldsValue != nil {
		objectValue, ok := fieldsValue.(*runtime.ObjectValue)
		if !ok {
			return "", "", fmt.Errorf("%s multipart must be object", name)
		}
		for key, fieldValue := range objectValue.Fields {
			switch typed := fieldValue.(type) {
			case *runtime.ArrayValue:
				for _, item := range typed.Elements {
					if err := writer.WriteField(key, item.String()); err != nil {
						return "", "", err
					}
				}
			default:
				if err := writer.WriteField(key, fieldValue.String()); err != nil {
					return "", "", err
				}
			}
		}
	}
	if filesValue != nil {
		arrayValue, ok := filesValue.(*runtime.ArrayValue)
		if !ok {
			return "", "", fmt.Errorf("%s files must be array", name)
		}
		for _, item := range arrayValue.Elements {
			fileValue, ok := item.(*runtime.ObjectValue)
			if !ok {
				return "", "", fmt.Errorf("%s files entries must be objects", name)
			}
			field, err := utils.RequireStringArg(name, fileValue.Fields["field"])
			if err != nil {
				return "", "", fmt.Errorf("%s file field: %w", name, err)
			}
			path, err := utils.RequireStringArg(name, fileValue.Fields["path"])
			if err != nil {
				return "", "", fmt.Errorf("%s file path: %w", name, err)
			}
			filename := filepath.Base(path)
			if nameValue, ok := fileValue.Fields["name"]; ok {
				filename, err = utils.RequireStringArg(name, nameValue)
				if err != nil {
					return "", "", err
				}
			}
			contentType := ""
			if contentTypeValue, ok := fileValue.Fields["contentType"]; ok {
				contentType, err = utils.RequireStringArg(name, contentTypeValue)
				if err != nil {
					return "", "", err
				}
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return "", "", err
			}
			var partWriter io.Writer
			if contentType != "" {
				header := make(textproto.MIMEHeader)
				header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, field, filename))
				header.Set("Content-Type", contentType)
				partWriter, err = writer.CreatePart(header)
				if err != nil {
					return "", "", err
				}
			} else {
				partWriter, err = writer.CreateFormFile(field, filename)
				if err != nil {
					return "", "", err
				}
			}
			if _, err := partWriter.Write(data); err != nil {
				return "", "", err
			}
		}
	}
	if err := writer.Close(); err != nil {
		return "", "", err
	}
	return buf.String(), writer.FormDataContentType(), nil
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
		Name:  "http.file",
		Arity: 1,
		Fn: func(args []runtime.Value) (runtime.Value, error) {
			name, err := utils.RequireStringArg("file", args[0])
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
		path, err := utils.RequireStringArg("cookie", pathValue)
		if err != nil {
			return nil, err
		}
		cookie.Path = path
	}
	if domainValue, ok := options.Fields["domain"]; ok {
		domain, err := utils.RequireStringArg("cookie", domainValue)
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
		header, err := utils.RequireStringArg(name, item)
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

func boolCount(v bool) int {
	if v {
		return 1
	}
	return 0
}
