package stdlib

import (
	"crypto/rand"
	"fmt"
	"strings"

	"icoo_lang/internal/runtime"
)

func loadStdUUIDModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.uuid",
		Path: "std.uuid",
		Exports: map[string]runtime.Value{
			"isValid": &runtime.NativeFunction{Name: "isValid", Arity: 1, Fn: uuidIsValid},
			"v4":      &runtime.NativeFunction{Name: "v4", Arity: 0, Fn: uuidV4},
		},
		Done: true,
	}
}

func uuidV4(args []runtime.Value) (runtime.Value, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return runtime.StringValue{Value: fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])}, nil
}

func uuidIsValid(args []runtime.Value) (runtime.Value, error) {
	text, err := requireStringArg("isValid", args[0])
	if err != nil {
		return nil, err
	}
	return runtime.BoolValue{Value: isValidUUIDV4(text)}, nil
}

func isValidUUIDV4(text string) bool {
	if len(text) != 36 {
		return false
	}
	for _, pos := range []int{8, 13, 18, 23} {
		if text[pos] != '-' {
			return false
		}
	}
	lower := strings.ToLower(text)
	if lower[14] != '4' {
		return false
	}
	if !strings.ContainsRune("89ab", rune(lower[19])) {
		return false
	}
	for i, ch := range lower {
		if ch == '-' {
			continue
		}
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
		if i == 14 || i == 19 {
			continue
		}
	}
	return true
}
