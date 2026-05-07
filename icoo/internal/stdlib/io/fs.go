package io

import (
	"fmt"
	golangio "io"
	"os"
	"path/filepath"
	"sort"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

// LoadStdIOFSModule 加载 std.io.fs 模块（文件系统操作）
func LoadStdIOFSModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.io.fs",
		Path: "std.io.fs",
		Exports: map[string]runtime.Value{
			"base":       &runtime.NativeFunction{Name: "base", Arity: 1, Fn: fsBase},
			"Copy":       &runtime.NativeFunction{Name: "Copy", Arity: 2, Fn: ioCopy},
			"copy":       &runtime.NativeFunction{Name: "copy", Arity: 2, Fn: ioCopy},
			"copyFile":   &runtime.NativeFunction{Name: "copyFile", Arity: 2, Fn: fsCopyFile},
			"dir":        &runtime.NativeFunction{Name: "dir", Arity: 1, Fn: fsDir},
			"exists":     &runtime.NativeFunction{Name: "exists", Arity: 1, Fn: fsExists},
			"join":       &runtime.NativeFunction{Name: "join", Arity: 2, Fn: fsJoin},
			"mkdir":      &runtime.NativeFunction{Name: "mkdir", Arity: 1, Fn: fsMkdir},
			"openReader": &runtime.NativeFunction{Name: "openReader", Arity: 1, Fn: ioOpenReader},
			"openWriter": &runtime.NativeFunction{Name: "openWriter", Arity: 1, Fn: ioOpenWriter},
			"readAll":    &runtime.NativeFunction{Name: "readAll", Arity: 1, Fn: ioReadAll},
			"readDir":    &runtime.NativeFunction{Name: "readDir", Arity: 1, Fn: fsReadDir},
			"readFile":   &runtime.NativeFunction{Name: "readFile", Arity: 1, Fn: fsReadFile},
			"remove":     &runtime.NativeFunction{Name: "remove", Arity: 1, Fn: fsRemove},
			"rename":     &runtime.NativeFunction{Name: "rename", Arity: 2, Fn: fsRename},
			"stat":       &runtime.NativeFunction{Name: "stat", Arity: 1, Fn: fsStat},
			"writeFile":  &runtime.NativeFunction{Name: "writeFile", Arity: 2, Fn: fsWriteFile},
		},
		Done: true,
	}
}

// fsReadFile 读取文件内容
func fsReadFile(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("readFile", args[0])
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(data)}, nil
}

// fsWriteFile 写入文件内容
func fsWriteFile(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("writeFile", args[0])
	if err != nil {
		return nil, err
	}
	content, err := utils.RequireStringArg("writeFile", args[1])
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

// fsExists 检查路径是否存在
func fsExists(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("exists", args[0])
	if err != nil {
		return nil, err
	}
	_, statErr := os.Stat(path)
	if statErr == nil {
		return runtime.BoolValue{Value: true}, nil
	}
	if os.IsNotExist(statErr) {
		return runtime.BoolValue{Value: false}, nil
	}
	return nil, statErr
}

// fsMkdir 创建目录
func fsMkdir(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("mkdir", args[0])
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

// fsRemove 删除文件
func fsRemove(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("remove", args[0])
	if err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

// fsReadDir 读取目录内容
func fsReadDir(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("readDir", args[0])
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	items := make([]runtime.Value, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		items = append(items, &runtime.ObjectValue{Fields: map[string]runtime.Value{
			"name":  runtime.StringValue{Value: name},
			"path":  runtime.StringValue{Value: filepath.Join(path, name)},
			"isDir": runtime.BoolValue{Value: entry.IsDir()},
		}})
	}
	return &runtime.ArrayValue{Elements: items}, nil
}

// fsStat 获取文件信息
func fsStat(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("stat", args[0])
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"name":    runtime.StringValue{Value: info.Name()},
		"size":    runtime.IntValue{Value: info.Size()},
		"isDir":   runtime.BoolValue{Value: info.IsDir()},
		"isFile":  runtime.BoolValue{Value: info.Mode().IsRegular()},
		"mode":    runtime.StringValue{Value: info.Mode().String()},
		"modTime": runtime.IntValue{Value: info.ModTime().Unix()},
	}}, nil
}

// fsRename 重命名文件
func fsRename(args []runtime.Value) (runtime.Value, error) {
	oldPath, err := utils.RequireStringArg("rename", args[0])
	if err != nil {
		return nil, err
	}
	newPath, err := utils.RequireStringArg("rename", args[1])
	if err != nil {
		return nil, err
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

// fsCopyFile 复制文件
func fsCopyFile(args []runtime.Value) (runtime.Value, error) {
	srcPath, err := utils.RequireStringArg("copyFile", args[0])
	if err != nil {
		return nil, err
	}
	dstPath, err := utils.RequireStringArg("copyFile", args[1])
	if err != nil {
		return nil, err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer src.Close()
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	defer dst.Close()
	if _, err := golangio.Copy(dst, src); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

// fsJoin 拼接路径
func fsJoin(args []runtime.Value) (runtime.Value, error) {
	left, err := utils.RequireStringArg("join", args[0])
	if err != nil {
		return nil, err
	}
	right, err := utils.RequireStringArg("join", args[1])
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: filepath.Join(left, right)}, nil
}

// fsBase 获取路径基础名
func fsBase(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("base", args[0])
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: filepath.Base(path)}, nil
}

// fsDir 获取路径目录部分
func fsDir(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("dir", args[0])
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: filepath.Dir(path)}, nil
}

// -- 流式 IO 操作（openReader / openWriter / copy / readAll） --

// ioHandle IO 句柄
type ioHandle struct {
	name   string
	file   *os.File
	reader golangio.Reader
	writer golangio.Writer
	closer golangio.Closer
}

// ioOpenReader 打开文件用于读取
func ioOpenReader(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("openReader", args[0])
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return newIOHandle("reader", file, file, nil, file), nil
}

// ioOpenWriter 打开文件用于写入
func ioOpenWriter(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("openWriter", args[0])
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	return newIOHandle("writer", file, nil, file, file), nil
}

// ioCopy 复制数据
func ioCopy(args []runtime.Value) (runtime.Value, error) {
	dstHandle, err := requireWriter("copy", args[0])
	if err != nil {
		return nil, err
	}
	srcHandle, err := requireReader("copy", args[1])
	if err != nil {
		return nil, err
	}
	written, err := golangio.Copy(dstHandle, srcHandle)
	if err != nil {
		return nil, err
	}
	return runtime.IntValue{Value: written}, nil
}

// ioReadAll 读取全部数据
func ioReadAll(args []runtime.Value) (runtime.Value, error) {
	reader, err := requireReader("readAll", args[0])
	if err != nil {
		return nil, err
	}
	data, err := golangio.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(data)}, nil
}

// newIOHandle 创建 IO 句柄对象
func newIOHandle(name string, file *os.File, reader golangio.Reader, writer golangio.Writer, closer golangio.Closer) *runtime.ObjectValue {
	handle := &ioHandle{name: name, file: file, reader: reader, writer: writer, closer: closer}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"close": &runtime.NativeFunction{Name: name + ".close", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			if handle.closer == nil {
				return runtime.NullValue{}, nil
			}
			if err := handle.closer.Close(); err != nil {
				return nil, err
			}
			handle.file = nil
			handle.reader = nil
			handle.writer = nil
			handle.closer = nil
			return runtime.NullValue{}, nil
		}},
		"kind": runtime.StringValue{Value: name},
		"raw":  &ioHandleValue{handle: handle},
	}}
}

// requireReader 要求参数为可读句柄
func requireReader(name string, v runtime.Value) (golangio.Reader, error) {
	handle, err := requireIOHandle(name, v)
	if err != nil {
		return nil, err
	}
	if handle.reader == nil {
		return nil, fmt.Errorf("%s expects readable io handle", name)
	}
	return handle.reader, nil
}

// requireWriter 要求参数为可写句柄
func requireWriter(name string, v runtime.Value) (golangio.Writer, error) {
	handle, err := requireIOHandle(name, v)
	if err != nil {
		return nil, err
	}
	if handle.writer == nil {
		return nil, fmt.Errorf("%s expects writable io handle", name)
	}
	return handle.writer, nil
}

// requireIOHandle 要求参数为 IO 句柄
func requireIOHandle(name string, v runtime.Value) (*ioHandle, error) {
	obj, ok := v.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s expects io handle", name)
	}
	raw, ok := obj.Fields["raw"].(*ioHandleValue)
	if !ok || raw.handle == nil {
		return nil, fmt.Errorf("%s expects io handle", name)
	}
	return raw.handle, nil
}

// ioHandleValue IO 句柄值类型
type ioHandleValue struct {
	handle *ioHandle
}

// Kind 返回值类型
func (v *ioHandleValue) Kind() runtime.ValueKind { return runtime.ObjectKind }

// String 返回字符串表示
func (v *ioHandleValue) String() string {
	if v == nil || v.handle == nil {
		return "<io nil>"
	}
	return "<io " + v.handle.name + ">"
}
