package stdnet

import (
	"fmt"
	"net"
	"strings"
	"time"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

type tcpConnBinding struct {
	conn net.Conn
}

type tcpServerBinding struct {
	listener net.Listener
}

type udpConnBinding struct {
	conn *net.UDPConn
}

type udpServerBinding struct {
	conn *net.UDPConn
}

type udpPacketBinding struct {
	conn *net.UDPConn
	addr *net.UDPAddr
}

func LoadStdNetSocketClientModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.net.socket.client",
		Path: "std.net.socket.client",
		Exports: map[string]runtime.Value{
			"connectTCP": &runtime.NativeFunction{Name: "connectTCP", Arity: 1, Fn: socketConnectTCP},
			"dialUDP":    &runtime.NativeFunction{Name: "dialUDP", Arity: 1, Fn: socketDialUDP},
		},
		Done: true,
	}
}

func LoadStdNetSocketServerModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.net.socket.server",
		Path: "std.net.socket.server",
		Exports: map[string]runtime.Value{
			"listenTCP": &runtime.NativeFunction{Name: "listenTCP", Arity: 1, CtxFn: socketListenTCP},
			"listenUDP": &runtime.NativeFunction{Name: "listenUDP", Arity: 1, CtxFn: socketListenUDP},
		},
		Done: true,
	}
}

func socketConnectTCP(args []runtime.Value) (runtime.Value, error) {
	opts, err := parseNetAddrOptions("connectTCP", args[0], 30*time.Second)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout("tcp", opts.Addr, opts.Timeout)
	if err != nil {
		return nil, err
	}
	return newTCPConnHandle(conn), nil
}

func socketDialUDP(args []runtime.Value) (runtime.Value, error) {
	opts, err := parseNetAddrOptions("dialUDP", args[0], 30*time.Second)
	if err != nil {
		return nil, err
	}
	remoteAddr, err := net.ResolveUDPAddr("udp", opts.Addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		return nil, err
	}
	return newUDPConnHandle(conn), nil
}

func socketListenTCP(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
	opts, handlerValue, err := parseSocketServerOptions("listenTCP", args[0])
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", opts.Addr)
	if err != nil {
		return nil, err
	}
	binding := &tcpServerBinding{listener: listener}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				_, err := ctx.CallDetached(handlerValue, []runtime.Value{newTCPConnHandle(conn)})
				if err != nil {
					_ = conn.Close()
				}
			}(conn)
		}
	}()
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"addr":  runtime.StringValue{Value: listener.Addr().String()},
		"close": &runtime.NativeFunction{Name: "close", Arity: 0, Fn: binding.close},
	}}, nil
}

func socketListenUDP(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
	opts, handlerValue, err := parseSocketServerOptions("listenUDP", args[0])
	if err != nil {
		return nil, err
	}
	addr, err := net.ResolveUDPAddr("udp", opts.Addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	binding := &udpServerBinding{conn: conn}
	go func() {
		buf := make([]byte, 64*1024)
		for {
			n, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			data := string(append([]byte(nil), buf[:n]...))
			packet := &runtime.ObjectValue{Fields: map[string]runtime.Value{
				"data":  runtime.StringValue{Value: data},
				"addr":  runtime.StringValue{Value: remoteAddr.String()},
				"reply": &runtime.NativeFunction{Name: "reply", Arity: 1, Fn: (&udpPacketBinding{conn: conn, addr: remoteAddr}).reply},
			}}
			go func() {
				_, _ = ctx.CallDetached(handlerValue, []runtime.Value{packet})
			}()
		}
	}()
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"addr":  runtime.StringValue{Value: conn.LocalAddr().String()},
		"close": &runtime.NativeFunction{Name: "close", Arity: 0, Fn: binding.close},
	}}, nil
}

func newTCPConnHandle(conn net.Conn) runtime.Value {
	binding := &tcpConnBinding{conn: conn}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"localAddr":  runtime.StringValue{Value: conn.LocalAddr().String()},
		"remoteAddr": runtime.StringValue{Value: conn.RemoteAddr().String()},
		"read":       &runtime.NativeFunction{Name: "read", Arity: 1, Fn: binding.read},
		"write":      &runtime.NativeFunction{Name: "write", Arity: 1, Fn: binding.write},
		"close":      &runtime.NativeFunction{Name: "close", Arity: 0, Fn: binding.close},
	}}
}

func newUDPConnHandle(conn *net.UDPConn) runtime.Value {
	binding := &udpConnBinding{conn: conn}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"localAddr":  runtime.StringValue{Value: conn.LocalAddr().String()},
		"remoteAddr": runtime.StringValue{Value: conn.RemoteAddr().String()},
		"read":       &runtime.NativeFunction{Name: "read", Arity: 1, Fn: binding.read},
		"write":      &runtime.NativeFunction{Name: "write", Arity: 1, Fn: binding.write},
		"close":      &runtime.NativeFunction{Name: "close", Arity: 0, Fn: binding.close},
	}}
}

func (binding *tcpConnBinding) read(args []runtime.Value) (runtime.Value, error) {
	size, err := requirePositiveIntArg("read", args[0])
	if err != nil {
		return nil, err
	}
	buf := make([]byte, size)
	n, err := binding.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(buf[:n])}, nil
}

func (binding *tcpConnBinding) write(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("write", args[0])
	if err != nil {
		return nil, err
	}
	n, err := binding.conn.Write([]byte(text))
	if err != nil {
		return nil, err
	}
	return runtime.IntValue{Value: int64(n)}, nil
}

func (binding *tcpConnBinding) close(args []runtime.Value) (runtime.Value, error) {
	if binding == nil || binding.conn == nil {
		return runtime.NullValue{}, nil
	}
	return runtime.NullValue{}, binding.conn.Close()
}

func (binding *tcpServerBinding) close(args []runtime.Value) (runtime.Value, error) {
	if binding == nil || binding.listener == nil {
		return runtime.NullValue{}, nil
	}
	return runtime.NullValue{}, binding.listener.Close()
}

func (binding *udpConnBinding) read(args []runtime.Value) (runtime.Value, error) {
	size, err := requirePositiveIntArg("read", args[0])
	if err != nil {
		return nil, err
	}
	buf := make([]byte, size)
	n, err := binding.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(buf[:n])}, nil
}

func (binding *udpConnBinding) write(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("write", args[0])
	if err != nil {
		return nil, err
	}
	n, err := binding.conn.Write([]byte(text))
	if err != nil {
		return nil, err
	}
	return runtime.IntValue{Value: int64(n)}, nil
}

func (binding *udpConnBinding) close(args []runtime.Value) (runtime.Value, error) {
	if binding == nil || binding.conn == nil {
		return runtime.NullValue{}, nil
	}
	return runtime.NullValue{}, binding.conn.Close()
}

func (binding *udpServerBinding) close(args []runtime.Value) (runtime.Value, error) {
	if binding == nil || binding.conn == nil {
		return runtime.NullValue{}, nil
	}
	return runtime.NullValue{}, binding.conn.Close()
}

func (binding *udpPacketBinding) reply(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("reply", args[0])
	if err != nil {
		return nil, err
	}
	n, err := binding.conn.WriteToUDP([]byte(text), binding.addr)
	if err != nil {
		return nil, err
	}
	return runtime.IntValue{Value: int64(n)}, nil
}

type netAddrOptions struct {
	Addr    string
	Timeout time.Duration
}

func parseNetAddrOptions(name string, v runtime.Value, fallback time.Duration) (*netAddrOptions, error) {
	obj, ok := v.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s expects options object", name)
	}
	addrValue, ok := obj.Fields["addr"].(runtime.StringValue)
	if !ok || strings.TrimSpace(addrValue.Value) == "" {
		return nil, fmt.Errorf("%s options require non-empty addr", name)
	}
	timeout, err := parseOptionalTimeout(obj, name, fallback)
	if err != nil {
		return nil, err
	}
	return &netAddrOptions{Addr: addrValue.Value, Timeout: timeout}, nil
}

func parseSocketServerOptions(name string, v runtime.Value) (*netAddrOptions, runtime.Value, error) {
	opts, err := parseNetAddrOptions(name, v, 0)
	if err != nil {
		return nil, nil, err
	}
	obj := v.(*runtime.ObjectValue)
	handlerValue, ok := obj.Fields["handler"]
	if !ok {
		return nil, nil, fmt.Errorf("%s options require handler", name)
	}
	if !isCallableValue(handlerValue) {
		return nil, nil, fmt.Errorf("%s handler must be callable", name)
	}
	return opts, handlerValue, nil
}

func requirePositiveIntArg(name string, v runtime.Value) (int, error) {
	value, ok := v.(runtime.IntValue)
	if !ok {
		return 0, fmt.Errorf("%s expects int argument", name)
	}
	if value.Value <= 0 {
		return 0, fmt.Errorf("%s expects positive int argument", name)
	}
	return int(value.Value), nil
}
