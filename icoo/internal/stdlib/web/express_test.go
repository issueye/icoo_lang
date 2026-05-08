package web

import (
	"testing"
	"time"

	"icoo_lang/internal/runtime"
)

func TestParseListenOptionsWithTimeouts(t *testing.T) {
	options, err := parseListenOptions(&runtime.ObjectValue{Fields: map[string]runtime.Value{
		"host":                runtime.StringValue{Value: "127.0.0.1"},
		"port":                runtime.IntValue{Value: 8080},
		"readTimeoutMs":       runtime.IntValue{Value: 1000},
		"readHeaderTimeoutMs": runtime.IntValue{Value: 2000},
		"writeTimeoutMs":      runtime.IntValue{Value: 3000},
		"idleTimeoutMs":       runtime.IntValue{Value: 4000},
	}})
	if err != nil {
		t.Fatalf("parse listen options: %v", err)
	}

	if options.addr != "127.0.0.1:8080" {
		t.Fatalf("unexpected addr: %s", options.addr)
	}
	if options.readTimeout != time.Second {
		t.Fatalf("unexpected read timeout: %v", options.readTimeout)
	}
	if options.readHeaderTimeout != 2*time.Second {
		t.Fatalf("unexpected read header timeout: %v", options.readHeaderTimeout)
	}
	if options.writeTimeout != 3*time.Second {
		t.Fatalf("unexpected write timeout: %v", options.writeTimeout)
	}
	if options.idleTimeout != 4*time.Second {
		t.Fatalf("unexpected idle timeout: %v", options.idleTimeout)
	}
}

func TestParseListenOptionsRejectsInvalidTimeoutType(t *testing.T) {
	_, err := parseListenOptions(&runtime.ObjectValue{Fields: map[string]runtime.Value{
		"addr":           runtime.StringValue{Value: "127.0.0.1:0"},
		"readTimeoutMs":  runtime.StringValue{Value: "bad"},
	}})
	if err == nil {
		t.Fatal("expected invalid timeout type error")
	}
	if err.Error() != "listen readTimeoutMs must be int" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseListenOptionsRejectsNegativeTimeout(t *testing.T) {
	_, err := parseListenOptions(&runtime.ObjectValue{Fields: map[string]runtime.Value{
		"addr":          runtime.StringValue{Value: "127.0.0.1:0"},
		"idleTimeoutMs": runtime.IntValue{Value: -1},
	}})
	if err == nil {
		t.Fatal("expected negative timeout error")
	}
	if err.Error() != "listen idleTimeoutMs must be non-negative" {
		t.Fatalf("unexpected error: %v", err)
	}
}
