package data

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"io"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

func LoadStdCompressModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.compress",
		Path: "std.compress",
		Exports: map[string]runtime.Value{
			"gzipCompress":   &runtime.NativeFunction{Name: "gzipCompress", Arity: 1, Fn: compressGzipCompress},
			"gzipDecompress": &runtime.NativeFunction{Name: "gzipDecompress", Arity: 1, Fn: compressGzipDecompress},
			"zlibCompress":   &runtime.NativeFunction{Name: "zlibCompress", Arity: 1, Fn: compressZlibCompress},
			"zlibDecompress": &runtime.NativeFunction{Name: "zlibDecompress", Arity: 1, Fn: compressZlibDecompress},
		},
		Done: true,
	}
}

func compressGzipCompress(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("gzipCompress", args[0])
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write([]byte(text)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: base64.StdEncoding.EncodeToString(buf.Bytes())}, nil
}

func compressGzipDecompress(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("gzipDecompress", args[0])
	if err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, err
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	plain, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(plain)}, nil
}

func compressZlibCompress(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("zlibCompress", args[0])
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	if _, err := writer.Write([]byte(text)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: base64.StdEncoding.EncodeToString(buf.Bytes())}, nil
}

func compressZlibDecompress(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("zlibDecompress", args[0])
	if err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, err
	}
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	plain, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(plain)}, nil
}
