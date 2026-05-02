package stdnet

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"icoo_lang/internal/runtime"
)

type sseClientBinding struct {
	resp   *http.Response
	reader *bufio.Reader
}

type sseConnectionBinding struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	closed  bool
	mu      sync.Mutex
}

func LoadStdNetSSEClientModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.net.sse.client",
		Path: "std.net.sse.client",
		Exports: map[string]runtime.Value{
			"connect": &runtime.NativeFunction{Name: "connect", Arity: 1, Fn: sseConnect},
			"request": &runtime.NativeFunction{Name: "request", Arity: 1, Fn: sseRequest},
		},
		Done: true,
	}
}

func LoadStdNetSSEServerModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.net.sse.server",
		Path: "std.net.sse.server",
		Exports: map[string]runtime.Value{
			"listen": &runtime.NativeFunction{Name: "listen", Arity: 1, CtxFn: sseListen},
		},
		Done: true,
	}
}

func sseConnect(args []runtime.Value) (runtime.Value, error) {
	opts, err := parseNetURLTimeoutOptions("connect", args[0])
	if err != nil {
		return nil, err
	}
	headers := map[string]string{}
	for key, value := range opts.Headers {
		headers[key] = value
	}
	if headers["Accept"] == "" {
		headers["Accept"] = "text/event-stream"
	}
	resp, err := doHTTPRoundTrip(&httpRequestOptions{
		Method:  opts.Method,
		URL:     opts.URL,
		Headers: headers,
		Body:    opts.Body,
		Timeout: opts.Timeout,
	})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("connect failed with status %d", resp.StatusCode)
	}
	binding := &sseClientBinding{resp: resp, reader: bufio.NewReader(resp.Body)}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"url":           runtime.StringValue{Value: opts.URL},
		"method":        runtime.StringValue{Value: opts.Method},
		"read":          &runtime.NativeFunction{Name: "read", Arity: 0, Fn: binding.read},
		"close":         &runtime.NativeFunction{Name: "close", Arity: 0, Fn: binding.close},
		"status":        runtime.IntValue{Value: int64(resp.StatusCode)},
		"header":        httpHeaderGetter(resp.Header),
		"hasHeader":     httpHasHeaderGetter(resp.Header),
		"headers":       httpHeadersToRuntime(resp.Header),
		"contentLength": runtime.IntValue{Value: resp.ContentLength},
	}}, nil
}

func sseRequest(args []runtime.Value) (runtime.Value, error) {
	opts, err := parseNetURLTimeoutOptions("request", args[0])
	if err != nil {
		return nil, err
	}
	headers := map[string]string{}
	for key, value := range opts.Headers {
		headers[key] = value
	}
	if headers["Accept"] == "" {
		headers["Accept"] = "text/event-stream"
	}
	resp, err := doHTTPRoundTrip(&httpRequestOptions{
		Method:  opts.Method,
		URL:     opts.URL,
		Headers: headers,
		Body:    opts.Body,
		Timeout: opts.Timeout,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	binding := &sseClientBinding{resp: resp, reader: bufio.NewReader(resp.Body)}
	events := make([]runtime.Value, 0, 16)
	for {
		event, err := binding.read(nil)
		if err != nil {
			return nil, err
		}
		if _, ok := event.(runtime.NullValue); ok {
			break
		}
		events = append(events, event)
	}

	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"url":           runtime.StringValue{Value: opts.URL},
		"method":        runtime.StringValue{Value: opts.Method},
		"status":        runtime.IntValue{Value: int64(resp.StatusCode)},
		"header":        httpHeaderGetter(resp.Header),
		"hasHeader":     httpHasHeaderGetter(resp.Header),
		"headers":       httpHeadersToRuntime(resp.Header),
		"contentLength": runtime.IntValue{Value: resp.ContentLength},
		"events":        &runtime.ArrayValue{Elements: events},
	}}, nil
}

func (binding *sseClientBinding) read(args []runtime.Value) (runtime.Value, error) {
	fields := map[string]runtime.Value{}
	dataLines := []string{}
	for {
		line, err := binding.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(fields) == 0 && len(dataLines) == 0 {
				return runtime.NullValue{}, nil
			}
			if err != io.EOF {
				return nil, err
			}
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if line == "" {
			if len(dataLines) > 0 {
				fields["data"] = runtime.StringValue{Value: strings.Join(dataLines, "\n")}
			}
			if _, ok := fields["event"]; !ok {
				fields["event"] = runtime.StringValue{Value: "message"}
			}
			return &runtime.ObjectValue{Fields: fields}, nil
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimPrefix(value, " ")
		switch name {
		case "event":
			fields["event"] = runtime.StringValue{Value: value}
		case "data":
			dataLines = append(dataLines, value)
		case "id":
			fields["id"] = runtime.StringValue{Value: value}
		case "retry":
			if n, err := strconv.ParseInt(value, 10, 64); err == nil {
				fields["retry"] = runtime.IntValue{Value: n}
			}
		}
		if err == io.EOF {
			return &runtime.ObjectValue{Fields: fields}, nil
		}
	}
}

func (binding *sseClientBinding) close(args []runtime.Value) (runtime.Value, error) {
	if binding == nil || binding.resp == nil || binding.resp.Body == nil {
		return runtime.NullValue{}, nil
	}
	return runtime.NullValue{}, binding.resp.Body.Close()
}

func sseListen(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
	obj, ok := args[0].(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("listen expects options object")
	}
	addrValue, ok := obj.Fields["addr"].(runtime.StringValue)
	if !ok || strings.TrimSpace(addrValue.Value) == "" {
		return nil, fmt.Errorf("listen options require non-empty addr")
	}
	path := "/events"
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
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		binding := &sseConnectionBinding{writer: w, flusher: flusher}
		reqValue, err := sseRequestToRuntime(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = ctx.CallDetached(handlerValue, []runtime.Value{newSSEConnectionHandle(binding), reqValue})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	binding := &httpServerBinding{server: &http.Server{Handler: mux}}
	go func() {
		_ = binding.server.Serve(listener)
	}()
	addr := listener.Addr().String()
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"addr":  runtime.StringValue{Value: addr},
		"url":   runtime.StringValue{Value: "http://" + addr + path},
		"close": &runtime.NativeFunction{Name: "close", Arity: 0, Fn: binding.close},
	}}, nil
}

func newSSEConnectionHandle(binding *sseConnectionBinding) runtime.Value {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"send":  &runtime.NativeFunction{Name: "send", Arity: 1, Fn: binding.send},
		"close": &runtime.NativeFunction{Name: "close", Arity: 0, Fn: binding.close},
	}}
}

func (binding *sseConnectionBinding) send(args []runtime.Value) (runtime.Value, error) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	if binding.closed {
		return runtime.NullValue{}, nil
	}
	text, err := formatSSEEvent(args[0])
	if err != nil {
		return nil, err
	}
	if _, err := io.WriteString(binding.writer, text); err != nil {
		return nil, err
	}
	binding.flusher.Flush()
	return runtime.NullValue{}, nil
}

func (binding *sseConnectionBinding) close(args []runtime.Value) (runtime.Value, error) {
	binding.mu.Lock()
	defer binding.mu.Unlock()
	binding.closed = true
	return runtime.NullValue{}, nil
}

func formatSSEEvent(v runtime.Value) (string, error) {
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
		return "", fmt.Errorf("send expects string or object")
	}
}

func sseRequestToRuntime(r *http.Request) (runtime.Value, error) {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"method":     runtime.StringValue{Value: r.Method},
		"url":        runtime.StringValue{Value: r.URL.String()},
		"path":       runtime.StringValue{Value: r.URL.Path},
		"headers":    httpHeadersToRuntime(r.Header),
		"host":       runtime.StringValue{Value: r.Host},
		"remoteAddr": runtime.StringValue{Value: r.RemoteAddr},
	}}, nil
}

func parseOptionalTimeout(obj *runtime.ObjectValue, name string, fallback time.Duration) (time.Duration, error) {
	if timeoutValue, ok := obj.Fields["timeoutMs"]; ok {
		intValue, ok := timeoutValue.(runtime.IntValue)
		if !ok {
			return 0, fmt.Errorf("%s timeoutMs must be int", name)
		}
		if intValue.Value < 0 {
			return 0, fmt.Errorf("%s timeoutMs must be non-negative", name)
		}
		return time.Duration(intValue.Value) * time.Millisecond, nil
	}
	return fallback, nil
}
